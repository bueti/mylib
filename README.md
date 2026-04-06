# mylib

Self-hosted ebook & PDF library manager. Single Go binary with an
embedded Svelte 5 web UI, in-browser EPUB/PDF reader, and OPDS 1.2
feed for e-reader apps.

## Features

- **Library scanning** — point at one or more directories; scans
  recursively with fsnotify for instant detection of new files
- **Metadata extraction** — EPUB (OPF + Dublin Core subjects), PDF
  (Info dict + first-page cover), filename heuristics fallback
- **Open Library enrichment** — auto-fills description, genres, series,
  covers for books with ISBN or title+author
- **Thematic browsing** — tag sidebar with book counts, browse-by-genre
  section, multi-tag OR filters, FTS5 full-text search
- **In-browser reader** — EPUB (epub.js, paginated, TOC, themes, font
  sizes) and PDF (native browser viewer). Cross-device reading progress
- **Multi-user** — bcrypt auth, Casbin RBAC (admin/reader roles),
  per-user collections and reading progress
- **OPDS 1.2** — root, recent, by-author, by-series, search feeds for
  KOReader, Moon+ Reader, etc.
- **Upload** — drag-and-drop from the browser, instant processing
- **Admin tools** — duplicate detection, metadata rescan, batch
  enrichment, book deletion (remove or delete file)
- **Single binary** — SvelteKit SPA embedded via `go:embed`, SQLite DB,
  no external services

See `PRD.md` for the full product spec.

## Quick start

```bash
# Build the SPA and Go binary
make build

# Run against a directory of books
MYLIB_LIBRARY_ROOTS=/path/to/books \
  MYLIB_ADMIN_USER=admin \
  MYLIB_ADMIN_PASSWORD=changeme \
  ./bin/mylib
```

Open:

- http://localhost:8080/ — web UI (login with admin credentials)
- http://localhost:8080/api/docs — API docs (Swagger UI)
- http://localhost:8080/api/openapi.json — OpenAPI 3.1 spec
- http://localhost:8080/opds — OPDS catalog (point KOReader here)

## Docker

```bash
docker pull ghcr.io/bueti/mylib:main

# Or use docker-compose
cp docker-compose.yml .
# Edit: set /path/to/your/books and MYLIB_ADMIN_PASSWORD
docker compose up -d
```

The image is ~30MB (alpine, non-root). Builds are pushed to GHCR on
every push to `main` and on `v*` tags.

## Configuration

All settings via env vars with `MYLIB_` prefix:

| Variable               | Default    | Description                             |
|------------------------|------------|-----------------------------------------|
| `MYLIB_LIBRARY_ROOTS`  | (required) | `:`-separated paths to scan             |
| `MYLIB_DATA_DIR`       | `./data`   | SQLite DB + cached covers               |
| `MYLIB_LISTEN`         | `:8080`    | HTTP listen address                     |
| `MYLIB_SCAN_INTERVAL`  | `10m`      | Periodic rescan cadence (`0` disables)  |
| `MYLIB_LOG_LEVEL`      | `info`     | `debug` / `info` / `warn` / `error`    |
| `MYLIB_ADMIN_USER`     |            | Bootstrap admin username (first run)    |
| `MYLIB_ADMIN_PASSWORD` |            | Bootstrap admin password (first run)    |

## Development

```bash
# Backend (auto-reload with air/reflex)
MYLIB_LIBRARY_ROOTS=./testdata \
  MYLIB_ADMIN_USER=admin MYLIB_ADMIN_PASSWORD=dev \
  go run ./cmd/mylib

# Frontend with live reload + proxy to localhost:8080
cd web && pnpm dev

# Regenerate TS API client from running server
make gen-api
```

Tests:

```bash
make test          # Go tests + svelte-check
go test ./...      # backend only
cd web && pnpm check
```

## Architecture

- **Go backend** (`cmd/mylib`, `internal/*`): Huma v2 on chi, Casbin RBAC,
  SQLite via `modernc.org/sqlite` with FTS5, `pdfcpu` for PDF metadata,
  Open Library client for enrichment, fsnotify watcher, SSE scan events.
- **Svelte 5 frontend** (`web/`): SvelteKit with `adapter-static`, runes
  (`$state`, `$effect`, `$props`), `openapi-fetch` typed client, epub.js
  reader with paginated flow + themes.
- **Single binary**: SPA compiled to `internal/webui/dist/` and embedded
  via `//go:embed`. No runtime asset dependencies.
