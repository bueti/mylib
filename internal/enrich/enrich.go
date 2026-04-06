package enrich

import (
	"context"
	"log/slog"
	"strings"

	"github.com/bueti/mylib/internal/covers"
	"github.com/bueti/mylib/internal/library"
)

// Enricher fills missing metadata on books by querying Open Library.
type Enricher struct {
	store  *library.Store
	client *OLClient
	covers *covers.Cache
}

// New builds an Enricher.
func New(store *library.Store, coverCache *covers.Cache) *Enricher {
	return &Enricher{
		store:  store,
		client: NewOLClient(),
		covers: coverCache,
	}
}

// EnrichBook looks up a single book on Open Library and fills any
// empty metadata fields. Returns true if anything changed.
func (e *Enricher) EnrichBook(ctx context.Context, bookID int64) (bool, error) {
	b, err := e.store.GetBook(ctx, bookID)
	if err != nil {
		return false, err
	}

	var result *LookupResult
	isbn := normalizeISBN(b.ISBN)
	if isbn != "" {
		result, err = e.client.LookupByISBN(ctx, isbn)
	}
	if result == nil && b.Title != "" {
		author := ""
		if len(b.Authors) > 0 {
			author = b.Authors[0].Name
		}
		result, err = e.client.LookupByTitleAuthor(ctx, b.Title, author)
	}
	if result == nil {
		if err != nil {
			return false, err
		}
		return false, nil
	}

	changed := false

	// Fill empty description.
	if b.Description == "" && result.Description != "" {
		b.Description = result.Description
		changed = true
	}

	// Normalize + merge subjects into tags (deduped against existing).
	normalized := NormalizeSubjects(result.Subjects)
	existingTags := make(map[string]struct{}, len(b.Tags))
	for _, t := range b.Tags {
		existingTags[strings.ToLower(t)] = struct{}{}
	}
	for _, s := range normalized {
		if _, ok := existingTags[strings.ToLower(s)]; !ok {
			b.Tags = append(b.Tags, s)
			existingTags[strings.ToLower(s)] = struct{}{}
			changed = true
		}
	}

	// Fill empty series.
	if b.SeriesName == "" && result.Series != "" {
		b.SeriesName = result.Series
		changed = true
	}

	// Fill empty publisher.
	if b.Publisher == "" && result.Publisher != "" {
		b.Publisher = result.Publisher
		changed = true
	}

	// Download cover if missing.
	if b.CoverPath == "" && result.CoverURL != "" && e.covers != nil {
		data, mime, err := e.client.DownloadCover(ctx, result.CoverURL)
		if err == nil && len(data) > 100 { // skip tiny error images
			rel, err := e.covers.Store(b.ContentHash, data, mime)
			if err == nil {
				b.CoverPath = rel
				changed = true
			}
		}
	}

	if !changed {
		return false, nil
	}
	if _, err := e.store.UpsertBook(ctx, b); err != nil {
		return false, err
	}
	slog.Info("enriched book", "id", bookID, "title", b.Title)
	return true, nil
}

// EnrichAll enriches all books missing description or tags.
// Returns count of books enriched and any final error.
func (e *Enricher) EnrichAll(ctx context.Context) (int, error) {
	books, _, err := e.store.ListBooks(ctx, library.BookFilter{Limit: 10_000})
	if err != nil {
		return 0, err
	}
	enriched := 0
	for _, b := range books {
		if b.Description != "" && len(b.Tags) > 0 {
			continue // already has metadata
		}
		ok, err := e.EnrichBook(ctx, b.ID)
		if err != nil {
			slog.Warn("enrich failed", "id", b.ID, "title", b.Title, "err", err)
			continue
		}
		if ok {
			enriched++
		}
	}
	return enriched, nil
}

// RunWorker drains bookIDs from the channel and enriches each.
// Intended to run as a goroutine in main.go.
func (e *Enricher) RunWorker(ctx context.Context, queue <-chan int64) {
	for {
		select {
		case <-ctx.Done():
			return
		case id, ok := <-queue:
			if !ok {
				return
			}
			if _, err := e.EnrichBook(ctx, id); err != nil {
				slog.Debug("async enrich failed", "id", id, "err", err)
			}
		}
	}
}

func normalizeISBN(s string) string {
	s = strings.TrimSpace(s)
	b := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c >= '0' && c <= '9') || c == 'x' || c == 'X' {
			b = append(b, c)
		}
	}
	if len(b) != 10 && len(b) != 13 {
		return ""
	}
	return string(b)
}
