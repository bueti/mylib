# mylib

Self-hosted ebook & PDF library manager. Single Go binary with an
embedded Svelte 5 web UI and an OPDS 1.2 feed for e-reader apps.

See `PRD.md` for the product spec. This is the v0.1 MVP (no auth,
no per-user reading progress yet).

## Quick start

```bash
# install frontend deps, build SPA, build Go binary
make build

# run against a directory of books
MYLIB_LIBRARY_ROOTS=/path/to/books ./bin/mylib
```

Then open:

- <http://localhost:8080/> — web UI
- <http://localhost:8080/api/docs> — API docs (Swagger UI)
- <http://localhost:8080/api/openapi.json> — OpenAPI 3.1 spec
- <http://localhost:8080/opds> — OPDS catalog root (point KOReader etc. here)

## Configuration

All settings come from env vars with the `MYLIB_` prefix:

| Variable                | Default    | Description                               |
|-------------------------|------------|-------------------------------------------|
| `MYLIB_LIBRARY_ROOTS`   | (required) | `:`-separated paths to scan               |
| `MYLIB_DATA_DIR`        | `./data`   | SQLite DB + cached covers                 |
| `MYLIB_LISTEN`          | `:8080`    | HTTP listen address                       |
| `MYLIB_SCAN_INTERVAL`   | `10m`      | Periodic rescan cadence (`0` disables)    |
| `MYLIB_LOG_LEVEL`       | `info`     | `debug` / `info` / `warn` / `error`       |

## Development

```bash
# backend only (auto-reloads require a tool like `air` or `reflex`)
MYLIB_LIBRARY_ROOTS=./testdata go run ./cmd/mylib

# frontend with live reload + proxy to localhost:8080
pnpm --dir web dev

# regenerate TS API client against the running server
make gen-api
```

Tests:

```bash
make test          # both Go and svelte-check
go test ./...      # backend only
pnpm --dir web check
```

## Architecture

- **Go backend** (`cmd/mylib`, `internal/*`): Huma v2 on chi, SQLite
  via `modernc.org/sqlite` with FTS5 for search, `pdfcpu` for PDF
  metadata, stdlib `archive/zip` for EPUB.
- **Svelte 5 frontend** (`web/`): SvelteKit with `adapter-static`,
  runes (`$state`, `$effect`, `$props`), `openapi-typescript` +
  `openapi-fetch` for the typed API client.
- **Build**: SPA is compiled to `internal/webui/dist/` and embedded
  into the Go binary via `//go:embed`, so `mylib` ships as a single
  binary with no runtime asset dependencies.
