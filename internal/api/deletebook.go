package api

import (
	"errors"
	"net/http"
	"os"

	"github.com/bueti/mylib/internal/library"
	"github.com/go-chi/chi/v5"
)

// registerDeleteBook wires the admin-only book deletion endpoint.
func registerDeleteBook(r chi.Router, store *library.Store) {
	r.With(RequireAuth(store), RequireAdmin).Delete("/api/books/{id}", func(w http.ResponseWriter, req *http.Request) {
		id := intParam(req, "id")
		if id <= 0 {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}
		b, err := store.GetBook(req.Context(), id)
		if errors.Is(err, library.ErrNotFound) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Check query param to decide whether to also remove the file.
		deleteFile := req.URL.Query().Get("delete_file") == "1"

		// Soft-delete from DB (removes from search + listings).
		if err := store.SoftDelete(req.Context(), id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Optionally remove the file from disk.
		if deleteFile {
			if err := os.Remove(b.Path); err != nil && !os.IsNotExist(err) {
				// Book is already soft-deleted from DB; warn but don't fail.
				writeJSON(w, http.StatusOK, map[string]any{
					"deleted":      true,
					"file_removed": false,
					"file_error":   err.Error(),
				})
				return
			}
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"deleted":      true,
			"file_removed": deleteFile,
		})
	})
}
