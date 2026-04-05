// Package opds renders OPDS 1.2 Atom feeds over the library. Clients
// like KOReader and Moon+ Reader consume this format directly.
package opds

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/bueti/mylib/internal/library"
	"github.com/go-chi/chi/v5"
)

const (
	mimeAtom        = "application/atom+xml;profile=opds-catalog;kind=navigation"
	mimeAcquisition = "application/atom+xml;profile=opds-catalog;kind=acquisition"
)

// Handler carries dependencies for the OPDS endpoints.
type Handler struct {
	Store *library.Store
}

// Mount registers OPDS routes on r at /opds.
func Mount(r chi.Router, h *Handler) {
	r.Get("/opds", h.root)
	r.Get("/opds/recent", h.recent)
	r.Get("/opds/authors", h.authorsNav)
	r.Get("/opds/authors/{id}", h.byAuthor)
	r.Get("/opds/series/{id}", h.bySeries)
	r.Get("/opds/search", h.search)
}

func (h *Handler) root(w http.ResponseWriter, r *http.Request) {
	feed := &feed{
		Xmlns:   "http://www.w3.org/2005/Atom",
		XmlnsDC: "http://purl.org/dc/terms/",
		XmlnsOP: "http://opds-spec.org/2010/catalog",
		ID:      "urn:mylib:catalog",
		Title:   "mylib",
		Updated: time.Now().UTC().Format(time.RFC3339),
		Links: []link{
			{Rel: "self", Href: "/opds", Type: mimeAtom},
			{Rel: "start", Href: "/opds", Type: mimeAtom},
		},
		Entries: []entry{
			navEntry("urn:mylib:recent", "Recently Added", "Newest books first", "/opds/recent", mimeAcquisition),
			navEntry("urn:mylib:authors", "Authors", "Browse by author", "/opds/authors", mimeAtom),
		},
	}
	writeFeed(w, mimeAtom, feed)
}

func (h *Handler) recent(w http.ResponseWriter, r *http.Request) {
	books, _, err := h.Store.ListBooks(r.Context(), library.BookFilter{Sort: "-added", Limit: 50})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeAcquisitionFeed(w, "urn:mylib:recent", "Recently Added", "/opds/recent", books)
}

func (h *Handler) authorsNav(w http.ResponseWriter, r *http.Request) {
	authors, err := h.Store.ListAuthors(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	entries := make([]entry, 0, len(authors))
	for _, a := range authors {
		entries = append(entries, navEntry(
			fmt.Sprintf("urn:mylib:author:%d", a.ID),
			a.Name, a.SortName,
			fmt.Sprintf("/opds/authors/%d", a.ID),
			mimeAcquisition,
		))
	}
	writeFeed(w, mimeAtom, &feed{
		Xmlns:   "http://www.w3.org/2005/Atom",
		XmlnsOP: "http://opds-spec.org/2010/catalog",
		ID:      "urn:mylib:authors",
		Title:   "Authors",
		Updated: time.Now().UTC().Format(time.RFC3339),
		Links: []link{
			{Rel: "self", Href: "/opds/authors", Type: mimeAtom},
			{Rel: "start", Href: "/opds", Type: mimeAtom},
			{Rel: "up", Href: "/opds", Type: mimeAtom},
		},
		Entries: entries,
	})
}

func (h *Handler) byAuthor(w http.ResponseWriter, r *http.Request) {
	id, ok := idParam(w, r)
	if !ok {
		return
	}
	books, _, err := h.Store.ListBooks(r.Context(), library.BookFilter{AuthorID: &id, Sort: "title", Limit: 500})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeAcquisitionFeed(w, fmt.Sprintf("urn:mylib:author:%d", id), "Books by author", fmt.Sprintf("/opds/authors/%d", id), books)
}

func (h *Handler) bySeries(w http.ResponseWriter, r *http.Request) {
	id, ok := idParam(w, r)
	if !ok {
		return
	}
	books, _, err := h.Store.ListBooks(r.Context(), library.BookFilter{SeriesID: &id, Sort: "title", Limit: 500})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeAcquisitionFeed(w, fmt.Sprintf("urn:mylib:series:%d", id), "Series", fmt.Sprintf("/opds/series/%d", id), books)
}

func (h *Handler) search(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	books, _, err := h.Store.ListBooks(r.Context(), library.BookFilter{Query: q, Limit: 100})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeAcquisitionFeed(w, "urn:mylib:search", "Search: "+q, "/opds/search?q="+q, books)
}

func writeAcquisitionFeed(w http.ResponseWriter, id, title, self string, books []*library.Book) {
	entries := make([]entry, 0, len(books))
	for _, b := range books {
		entries = append(entries, bookEntry(b))
	}
	writeFeed(w, mimeAcquisition, &feed{
		Xmlns:   "http://www.w3.org/2005/Atom",
		XmlnsDC: "http://purl.org/dc/terms/",
		XmlnsOP: "http://opds-spec.org/2010/catalog",
		ID:      id,
		Title:   title,
		Updated: time.Now().UTC().Format(time.RFC3339),
		Links: []link{
			{Rel: "self", Href: self, Type: mimeAcquisition},
			{Rel: "start", Href: "/opds", Type: mimeAtom},
			{Rel: "up", Href: "/opds", Type: mimeAtom},
		},
		Entries: entries,
	})
}

func bookEntry(b *library.Book) entry {
	e := entry{
		ID:      fmt.Sprintf("urn:mylib:book:%d", b.ID),
		Title:   b.Title,
		Updated: b.AddedAt.UTC().Format(time.RFC3339),
		Summary: b.Description,
	}
	for _, a := range b.Authors {
		e.Authors = append(e.Authors, author{Name: a.Name})
	}
	if b.Language != "" {
		e.Language = b.Language
	}
	if b.Publisher != "" {
		e.Publisher = b.Publisher
	}
	if b.PublishedAt != "" {
		e.Issued = b.PublishedAt
	}
	// Acquisition link (download).
	e.Links = append(e.Links, link{
		Rel:  "http://opds-spec.org/acquisition",
		Href: fmt.Sprintf("/api/books/%d/file", b.ID),
		Type: acquisitionMIME(b.Format),
	})
	// Cover link (if present).
	if b.CoverPath != "" {
		e.Links = append(e.Links, link{
			Rel:  "http://opds-spec.org/image",
			Href: fmt.Sprintf("/api/books/%d/cover", b.ID),
			Type: "image/jpeg",
		})
		e.Links = append(e.Links, link{
			Rel:  "http://opds-spec.org/image/thumbnail",
			Href: fmt.Sprintf("/api/books/%d/cover", b.ID),
			Type: "image/jpeg",
		})
	}
	return e
}

func navEntry(id, title, summary, href, linkType string) entry {
	return entry{
		ID:      id,
		Title:   title,
		Summary: summary,
		Updated: time.Now().UTC().Format(time.RFC3339),
		Links: []link{
			{Rel: "subsection", Href: href, Type: linkType},
		},
	}
}

func acquisitionMIME(format string) string {
	switch format {
	case "epub":
		return "application/epub+zip"
	case "pdf":
		return "application/pdf"
	case "mobi":
		return "application/x-mobipocket-ebook"
	case "azw3":
		return "application/vnd.amazon.ebook"
	default:
		return "application/octet-stream"
	}
}

func idParam(w http.ResponseWriter, r *http.Request) (int64, bool) {
	raw := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return 0, false
	}
	return id, true
}

func writeFeed(w http.ResponseWriter, mimeType string, f *feed) {
	w.Header().Set("Content-Type", mimeType+"; charset=utf-8")
	w.Write([]byte(xml.Header))
	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")
	if err := enc.Encode(f); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	enc.Flush()
}

// --- XML structs ---

type feed struct {
	XMLName xml.Name `xml:"feed"`
	Xmlns   string   `xml:"xmlns,attr"`
	XmlnsDC string   `xml:"xmlns:dc,attr,omitempty"`
	XmlnsOP string   `xml:"xmlns:opds,attr,omitempty"`
	ID      string   `xml:"id"`
	Title   string   `xml:"title"`
	Updated string   `xml:"updated"`
	Links   []link   `xml:"link"`
	Entries []entry  `xml:"entry"`
}

type entry struct {
	ID        string   `xml:"id"`
	Title     string   `xml:"title"`
	Updated   string   `xml:"updated"`
	Summary   string   `xml:"summary,omitempty"`
	Language  string   `xml:"dc:language,omitempty"`
	Publisher string   `xml:"dc:publisher,omitempty"`
	Issued    string   `xml:"dc:issued,omitempty"`
	Authors   []author `xml:"author"`
	Links     []link   `xml:"link"`
}

type author struct {
	Name string `xml:"name"`
}

type link struct {
	Rel  string `xml:"rel,attr"`
	Href string `xml:"href,attr"`
	Type string `xml:"type,attr,omitempty"`
}
