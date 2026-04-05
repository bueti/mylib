// Package webui serves the embedded SvelteKit SPA.
package webui

import (
	"bytes"
	"embed"
	"errors"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

//go:embed all:dist
var distFS embed.FS

// Handler returns an http.Handler that serves the embedded SvelteKit
// dist with an SPA fallback: unknown paths resolve to index.html so
// client-side routing works.
func Handler() http.Handler {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "web UI not built: run `pnpm --dir web dist`", http.StatusNotFound)
		})
	}
	index, err := fs.ReadFile(sub, "index.html")
	if err != nil {
		// index.html missing → the SPA wasn't built. Return a helpful
		// message at every path so the user knows how to fix it.
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "web UI not built: run `pnpm --dir web dist`", http.StatusNotFound)
		})
	}
	files := http.FileServer(http.FS(sub))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try to serve a real file; otherwise fall back to index.html.
		p := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if p == "" {
			p = "index.html"
		}
		if _, err := fs.Stat(sub, p); err == nil {
			files.ServeHTTP(w, r)
			return
		}
		if !errors.Is(err, fs.ErrNotExist) {
			// unexpected; fall through to index
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		http.ServeContent(w, r, "index.html", staticModTime, bytes.NewReader(index))
	})
}
