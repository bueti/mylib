// Package api wires the Huma HTTP handlers for mylib.
package api

import (
	"net/http"

	"github.com/bueti/mylib/internal/covers"
	"github.com/bueti/mylib/internal/library"
	"github.com/bueti/mylib/internal/scanner"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// Deps bundles the handler dependencies.
type Deps struct {
	Store   *library.Store
	Scanner *scanner.Scanner
	Covers  *covers.Cache
}

// NewRouter returns a chi router with the mylib HTTP API mounted at /api
// and raw cover/file downloads mounted at their own paths.
func NewRouter(deps Deps) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)

	// Mount the huma API under /api by creating a sub-router and using
	// Huma's path prefix support so all operation paths are rewritten.
	apiRouter := chi.NewRouter()
	humaCfg := huma.DefaultConfig("mylib", "0.1.0")
	humaCfg.OpenAPIPath = "/openapi"
	humaCfg.DocsPath = "/docs"
	humaCfg.SchemasPath = "/schemas"
	api := humachi.New(apiRouter, humaCfg)
	r.Mount("/api", apiRouter)

	registerBooks(api, deps)
	registerTaxonomy(api, deps)
	registerScan(api, deps)

	return r
}
