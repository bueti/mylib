// Package api wires the Huma HTTP handlers for mylib.
package api

import (
	"net/http"

	"github.com/bueti/mylib/internal/covers"
	"github.com/bueti/mylib/internal/enrich"
	"github.com/bueti/mylib/internal/library"
	"github.com/bueti/mylib/internal/scanner"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// Deps bundles the handler dependencies.
type Deps struct {
	Store    *library.Store
	Scanner  *scanner.Scanner
	Covers   *covers.Cache
	Enricher *enrich.Enricher
}

// NewRouter returns a chi router exposing the mylib HTTP API. All
// routes (huma-managed and raw) use absolute /api/... paths. The
// caller mounts this router at "/".
func NewRouter(deps Deps) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	// Attach user (if any) to every request. Individual handlers
	// decide whether to require it.
	r.Use(OptionalAuth(deps.Store))

	humaCfg := huma.DefaultConfig("mylib", "0.2.0")
	humaCfg.OpenAPIPath = "/api/openapi"
	humaCfg.DocsPath = "/api/docs"
	humaCfg.SchemasPath = "/api/schemas"
	api := humachi.New(r, humaCfg)

	registerAuth(r, deps.Store)
	registerUsers(r, deps.Store)
	registerBooks(api, deps)
	registerTaxonomy(api, deps)
	registerScan(api, deps)
	registerProgress(api, deps)
	registerCollections(api, deps)
	registerFileRoutes(r, deps)
	registerSSE(r, deps)
	registerAdmin(r, deps.Store)
	if deps.Enricher != nil {
		registerEnrich(r, deps.Store, deps.Enricher)
	}

	return r
}
