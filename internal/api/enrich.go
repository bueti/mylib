package api

import (
	"net/http"

	"github.com/bueti/mylib/internal/authz"
	"github.com/bueti/mylib/internal/enrich"
	"github.com/bueti/mylib/internal/library"
	"github.com/go-chi/chi/v5"
)

// registerEnrich wires metadata enrichment endpoints.
func registerEnrich(r chi.Router, store *library.Store, enricher *enrich.Enricher, az *authz.Authorizer) {
	// Any authenticated user can enrich a single book.
	r.With(RequireAuth(store)).Post("/api/books/{id}/enrich", func(w http.ResponseWriter, req *http.Request) {
		id := intParam(req, "id")
		if id <= 0 {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}
		changed, err := enricher.EnrichBook(req.Context(), id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		b, err := store.GetBook(req.Context(), id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		_ = changed
		writeJSON(w, http.StatusOK, toBookDTO(b))
	})

	// Admin-only batch enrichment.
	r.With(RequireAuth(store), Authorize(az, "admin", "access")).Post("/api/admin/enrich-all", func(w http.ResponseWriter, req *http.Request) {
		n, err := enricher.EnrichAll(req.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"enriched": n})
	})
}
