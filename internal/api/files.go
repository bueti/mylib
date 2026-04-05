package api

import (
	"errors"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/bueti/mylib/internal/library"
	"github.com/go-chi/chi/v5"
)

// registerFileRoutes wires raw streaming endpoints on the chi router for
// cover images and book file downloads. These live outside the Huma API
// so we can use http.ServeContent for Range + ETag semantics.
func registerFileRoutes(r chi.Router, d Deps) {
	r.Get("/api/books/{id}/cover", func(w http.ResponseWriter, req *http.Request) {
		id, ok := pathID(w, req)
		if !ok {
			return
		}
		b, err := d.Store.GetBook(req.Context(), id)
		if errors.Is(err, library.ErrNotFound) || (b != nil && b.CoverPath == "") {
			http.Error(w, "no cover", http.StatusNotFound)
			return
		}
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		abs := d.Covers.AbsPath(b.CoverPath)
		f, err := os.Open(abs)
		if err != nil {
			http.Error(w, "cover missing on disk", http.StatusNotFound)
			return
		}
		defer f.Close()
		st, err := f.Stat()
		if err != nil {
			http.Error(w, "stat failed", http.StatusInternalServerError)
			return
		}
		// ETag keyed on content hash so clients cache aggressively.
		w.Header().Set("ETag", `"`+b.ContentHash+`"`)
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		if ct := mime.TypeByExtension(filepath.Ext(abs)); ct != "" {
			w.Header().Set("Content-Type", ct)
		}
		http.ServeContent(w, req, filepath.Base(abs), st.ModTime(), f)
	})

	r.Get("/api/books/{id}/file", func(w http.ResponseWriter, req *http.Request) {
		id, ok := pathID(w, req)
		if !ok {
			return
		}
		b, err := d.Store.GetBook(req.Context(), id)
		if errors.Is(err, library.ErrNotFound) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		f, err := os.Open(b.Path)
		if err != nil {
			http.Error(w, "file missing on disk", http.StatusNotFound)
			return
		}
		defer f.Close()
		st, err := f.Stat()
		if err != nil {
			http.Error(w, "stat failed", http.StatusInternalServerError)
			return
		}
		w.Header().Set("ETag", `"`+b.ContentHash+`"`)
		// Browser viewers (PDF iframe, epub.js fetch) need the bytes
		// served inline; the default for downloads is attachment.
		disposition := "attachment"
		if req.URL.Query().Get("inline") == "1" {
			disposition = "inline"
		}
		w.Header().Set("Content-Disposition",
			disposition+`; filename="`+safeFilename(b.Title, b.Format)+`"`)
		w.Header().Set("Content-Type", contentTypeFor(b.Format))
		http.ServeContent(w, req, filepath.Base(b.Path), st.ModTime(), f)
	})
}

// pathID extracts an integer {id} URL parameter, writing 400 on failure.
func pathID(w http.ResponseWriter, req *http.Request) (int64, bool) {
	raw := chi.URLParam(req, "id")
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return 0, false
	}
	return id, true
}

func contentTypeFor(format string) string {
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

// safeFilename produces a download filename without path separators.
func safeFilename(title, format string) string {
	out := make([]byte, 0, len(title)+5)
	for i := 0; i < len(title); i++ {
		c := title[i]
		switch c {
		case '/', '\\', '"', '\n', '\r':
			out = append(out, '_')
		default:
			out = append(out, c)
		}
	}
	return string(out) + "." + format
}
