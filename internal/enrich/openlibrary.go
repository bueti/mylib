// Package enrich queries external metadata providers (Open Library) to
// fill in missing fields on books — description, subjects/tags, series,
// and covers.
package enrich

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// olEdition is the shape returned by /isbn/{isbn}.json.
type olEdition struct {
	Title      string   `json:"title"`
	Publishers []string `json:"publishers"`
	PublishDate string  `json:"publish_date"`
	Subjects   []string `json:"subjects"`
	Covers     []int    `json:"covers"`
	Works      []struct {
		Key string `json:"key"` // e.g. "/works/OL45883W"
	} `json:"works"`
}

// olWork is the shape returned by /works/{key}.json.
type olWork struct {
	Title    string   `json:"title"`
	Subjects []string `json:"subjects"`
	// Description can be a string or {"type":"/type/text","value":"..."}.
	Description json.RawMessage `json:"description"`
	Series      []string        `json:"series"`
}

func (w *olWork) descriptionText() string {
	if len(w.Description) == 0 {
		return ""
	}
	// Try plain string first.
	var s string
	if json.Unmarshal(w.Description, &s) == nil {
		return s
	}
	// Try object with value field.
	var obj struct{ Value string }
	if json.Unmarshal(w.Description, &obj) == nil {
		return obj.Value
	}
	return ""
}

// olSearchResult is returned by /search.json.
type olSearchResult struct {
	Docs []struct {
		Key     string   `json:"key"` // e.g. "/works/OL45883W"
		Subject []string `json:"subject"`
	} `json:"docs"`
}

// OLClient wraps the Open Library API with a built-in rate limiter.
type OLClient struct {
	http    *http.Client
	limiter <-chan time.Time // 1 req/sec
}

// NewOLClient builds a rate-limited Open Library client.
func NewOLClient() *OLClient {
	return &OLClient{
		http:    &http.Client{Timeout: 10 * time.Second},
		limiter: time.Tick(time.Second), //nolint:staticcheck // acceptable for long-lived client
	}
}

// LookupResult holds everything we got from Open Library.
type LookupResult struct {
	Description string
	Subjects    []string
	Series      string
	CoverURL    string // https://covers.openlibrary.org/...
	Publisher   string
	PublishDate string
}

// LookupByISBN queries by ISBN, then fetches the associated work.
func (c *OLClient) LookupByISBN(ctx context.Context, isbn string) (*LookupResult, error) {
	<-c.limiter
	edition, err := c.fetchEdition(ctx, isbn)
	if err != nil {
		return nil, err
	}
	result := &LookupResult{
		Subjects: dedup(edition.Subjects),
	}
	if len(edition.Publishers) > 0 {
		result.Publisher = edition.Publishers[0]
	}
	if edition.PublishDate != "" {
		result.PublishDate = edition.PublishDate
	}
	if len(edition.Covers) > 0 && edition.Covers[0] > 0 {
		result.CoverURL = fmt.Sprintf("https://covers.openlibrary.org/b/id/%d-L.jpg", edition.Covers[0])
	}
	// Fetch the work for description + more subjects.
	if len(edition.Works) > 0 {
		work, err := c.fetchWork(ctx, edition.Works[0].Key)
		if err == nil {
			if desc := work.descriptionText(); desc != "" {
				result.Description = desc
			}
			result.Subjects = dedup(append(result.Subjects, work.Subjects...))
			if len(work.Series) > 0 {
				result.Series = work.Series[0]
			}
		}
	}
	return result, nil
}

// LookupByTitleAuthor falls back to search when no ISBN is available.
func (c *OLClient) LookupByTitleAuthor(ctx context.Context, title, author string) (*LookupResult, error) {
	<-c.limiter
	q := url.Values{}
	q.Set("title", title)
	if author != "" {
		q.Set("author", author)
	}
	q.Set("limit", "1")
	req, err := http.NewRequestWithContext(ctx, "GET",
		"https://openlibrary.org/search.json?"+q.Encode(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "mylib/0.2 (ebook library manager)")
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("OL search: %s", resp.Status)
	}
	var sr olSearchResult
	if err := json.NewDecoder(resp.Body).Decode(&sr); err != nil {
		return nil, err
	}
	if len(sr.Docs) == 0 {
		return nil, fmt.Errorf("no results for %q by %q", title, author)
	}
	doc := sr.Docs[0]
	result := &LookupResult{
		Subjects: dedup(doc.Subject),
	}
	// Fetch the work for description.
	if doc.Key != "" {
		work, err := c.fetchWork(ctx, doc.Key)
		if err == nil {
			result.Description = work.descriptionText()
			result.Subjects = dedup(append(result.Subjects, work.Subjects...))
			if len(work.Series) > 0 {
				result.Series = work.Series[0]
			}
		}
	}
	return result, nil
}

func (c *OLClient) fetchEdition(ctx context.Context, isbn string) (*olEdition, error) {
	req, err := http.NewRequestWithContext(ctx, "GET",
		"https://openlibrary.org/isbn/"+isbn+".json", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "mylib/0.2 (ebook library manager)")
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == 404 {
		return nil, fmt.Errorf("ISBN %s not found on Open Library", isbn)
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("OL edition: %s", resp.Status)
	}
	var ed olEdition
	if err := json.NewDecoder(resp.Body).Decode(&ed); err != nil {
		return nil, err
	}
	return &ed, nil
}

func (c *OLClient) fetchWork(ctx context.Context, key string) (*olWork, error) {
	<-c.limiter
	req, err := http.NewRequestWithContext(ctx, "GET",
		"https://openlibrary.org"+key+".json", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "mylib/0.2 (ebook library manager)")
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("OL work: %s", resp.Status)
	}
	var w olWork
	if err := json.NewDecoder(resp.Body).Decode(&w); err != nil {
		return nil, err
	}
	return &w, nil
}

// DownloadCover fetches the cover image bytes from the given URL.
func (c *OLClient) DownloadCover(ctx context.Context, coverURL string) ([]byte, string, error) {
	<-c.limiter
	req, err := http.NewRequestWithContext(ctx, "GET", coverURL, nil)
	if err != nil {
		return nil, "", err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, "", fmt.Errorf("cover download: %s", resp.Status)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}
	ct := resp.Header.Get("Content-Type")
	if ct == "" {
		ct = http.DetectContentType(data)
	}
	return data, ct, nil
}

// dedup returns unique, non-empty, trimmed strings, preserving order.
// At most 15 subjects are kept (Open Library can return hundreds).
func dedup(xs []string) []string {
	seen := make(map[string]struct{}, len(xs))
	var out []string
	for _, s := range xs {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		lower := strings.ToLower(s)
		if _, ok := seen[lower]; ok {
			continue
		}
		seen[lower] = struct{}{}
		out = append(out, s)
		if len(out) >= 15 {
			break
		}
	}
	return out
}
