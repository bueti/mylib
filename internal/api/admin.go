package api

import (
	"net/http"

	"github.com/bueti/mylib/internal/authz"
	"github.com/bueti/mylib/internal/enrich"
	"github.com/bueti/mylib/internal/library"
	"github.com/bueti/mylib/internal/scanner"
	"github.com/go-chi/chi/v5"
)

// DuplicateBookDTO is a small projection of a Book suitable for the
// duplicates view.
type DuplicateBookDTO struct {
	ID        int64  `json:"id"`
	Title     string `json:"title"`
	Path      string `json:"path"`
	Format    string `json:"format"`
	SizeBytes int64  `json:"size_bytes"`
	ISBN      string `json:"isbn,omitempty"`
	HasCover  bool   `json:"has_cover"`
}

// DuplicateGroupDTO wraps a group of suspected duplicates.
type DuplicateGroupDTO struct {
	Reason string             `json:"reason"` // "isbn" or "title"
	Key    string             `json:"key"`
	Books  []DuplicateBookDTO `json:"books"`
}

// registerAdmin wires admin-only maintenance endpoints.
func registerAdmin(r chi.Router, store *library.Store, sc *scanner.Scanner, az *authz.Authorizer) {
	r.With(RequireAuth(store), Authorize(az, "admin", "access")).Get("/api/admin/duplicates", func(w http.ResponseWriter, req *http.Request) {
		groups, err := store.FindDuplicates(req.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		out := make([]DuplicateGroupDTO, 0, len(groups))
		for _, g := range groups {
			dto := DuplicateGroupDTO{Reason: g.Reason, Key: g.Key}
			for _, b := range g.Books {
				dto.Books = append(dto.Books, DuplicateBookDTO{
					ID: b.ID, Title: b.Title, Path: b.Path, Format: b.Format,
					SizeBytes: b.SizeBytes, ISBN: b.ISBN, HasCover: b.CoverPath != "",
				})
			}
			out = append(out, dto)
		}
		writeJSON(w, http.StatusOK, map[string]any{"groups": out})
	})

	r.With(RequireAuth(store), Authorize(az, "admin", "access")).Post("/api/admin/normalize-tags", func(w http.ResponseWriter, req *http.Request) {
		n, err := store.RenormalizeTags(req.Context(), enrich.NormalizeSubjects)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		orphans, _ := store.CleanOrphanTags(req.Context())
		writeJSON(w, http.StatusOK, map[string]any{"updated": n, "orphan_tags_removed": orphans})
	})

	r.With(RequireAuth(store), Authorize(az, "admin", "access")).Post("/api/admin/rescan-metadata", func(w http.ResponseWriter, req *http.Request) {
		n, err := sc.ForceRescan(req.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"updated": n})
	})
}
