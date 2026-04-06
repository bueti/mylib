# mylib — Personal Ebook & PDF Library

## 1. Overview

A self-hosted library manager for ebooks (EPUB, MOBI, AZW3) and PDFs, in the
spirit of [Audiobookshelf](https://www.audiobookshelf.org/) but focused on
reading formats rather than audio. Users point the server at one or more
directories on disk; the server scans, extracts metadata, and exposes a
browsable, searchable collection via a typed HTTP API with generated clients.

Ships as a single Go binary with an embedded Svelte 5 web UI and an OPDS 1.2
feed for e-reader apps.

## 2. Goals

- **Self-hosted, single binary.** Runs locally or on a home server with minimal
  configuration. SQLite by default; no external services required.
- **Non-destructive.** Source files on disk are the source of truth. The server
  never moves, renames, or rewrites files unless explicitly told to.
- **Typed API first.** Every endpoint is described in OpenAPI 3.1, generated
  automatically by Huma. TypeScript client generated from the spec.
- **Fast library operations.** Scans of 10k+ files complete in seconds on
  re-scan; full-text metadata search is sub-100ms.
- **Multi-user with progress tracking.** Each user has their own reading
  progress, collections, and preferences over a shared library.

## 3. Non-Goals

- Audiobook support (that's what Audiobookshelf is for).
- Social features (sharing, ratings, recommendations).
- Cloud sync of user libraries across instances.
- DRM handling. Users are responsible for their own files.
- Migrating to Postgres. SQLite remains the only supported store.

## 4. Users & Use Cases

**Primary user:** a technical home-server operator with a personal ebook
collection spread across folders, wanting a unified catalog.

**Use cases:**

1. Point the server at `~/Books` and browse by author/series/genre in a web UI.
2. Search "Le Guin" and download an EPUB to a Kobo over the local network.
3. Open an EPUB in the browser reader, close it, reopen later on another device
   and resume at the same page.
4. Subscribe to the library's OPDS feed from KOReader on an e-ink device.
5. Tag a subset of books as "2026 reading list" and filter by that tag.
6. Upload new books directly from the browser without SSH access.
7. Browse the library by genre (Fantasy, Science Fiction, Philosophy…) via an
   auto-populated tag sidebar.

## 5. Features — Shipped

### 5.1 Core library (v0.1)

- **Library scanning.** Recursive directory walk; detects new, modified, and
  deleted files. Incremental rescans using mtime + size. Content-hash-based
  dedup (SHA-256 of first/last 64KB + length). Soft-deletes for removed files.
- **Metadata extraction.**
  - EPUB: parse OPF for title, authors, series (calibre:series), publisher,
    language, ISBN, description, `dc:subject` genres, cover image.
  - PDF: pdfcpu Info dictionary for title/author/keywords. First-page image
    extraction for cover thumbnails.
  - MOBI/AZW3: filename-heuristic fallback ("Author - Title.ext").
- **Metadata editing.** `PATCH /api/books/{id}` for title, subtitle, authors,
  series, tags, description, publisher, language, ISBN. Edit form on detail page.
- **Browse & search.** By author, series, tag, format, collection, date added.
  Full-text search over title/author/description/tags (SQLite FTS5). Multi-tag
  OR filtering. Sort by title, date added, reverse.
- **Thematic browsing.** Tag sidebar with book counts, sorted by frequency.
  "Browse by genre" card section on the home page. Tags auto-populated from
  embedded EPUB subjects and Open Library enrichment, normalized to canonical
  genres (e.g. "Fiction, fantasy, general" → "Fantasy").
- **Collections.** Per-user named groupings. CRUD API + UI for list, view,
  add-to-collection from book detail page with inline new-collection prompt.
- **Download.** Direct file download with Range + ETag support. Content-type
  headers for epub/pdf/mobi/azw3.
- **OPDS 1.2.** Root navigation, recent, by-author, by-series, search feeds.
  Acquisition links to /api/books/{id}/file. Cover thumbnails.
- **Covers.** Extracted from EPUB (cover-image property or meta cover id) and
  PDF (largest first-page image). Cached on disk sharded by content hash.
  Served with immutable Cache-Control + ETag.

### 5.2 Auth & multi-user (v0.2)

- **Local auth.** Username + bcrypt password. Server-side sessions (256-bit
  random hex token) stored in SQLite. HttpOnly + SameSite=Lax cookies.
  Admin bootstrap via `MYLIB_ADMIN_USER`/`MYLIB_ADMIN_PASSWORD` env vars on
  first startup.
- **Casbin RBAC.** Policy-driven authorization replacing ad-hoc middleware.
  Two roles: admin (full access) and reader (can read/edit/upload/enrich but
  not delete books or manage users). `Authorize(az, resource, action)`
  middleware. `GET /api/auth/permissions` returns the user's effective
  permission set so the frontend conditionally shows/hides UI elements.
- **Server-side reading progress.** Per-user, per-book CFI (EPUB) or page
  (PDF) position, percent, finished flag, theme/font-size preferences.
  Debounced save (5s) + sendBeacon on unload. Cross-device resume.
  "Continue reading" row on the home page.
  localStorage migration from v0.1 on first login.
- **Admin user management.** `POST /api/users` (create), `DELETE /api/users/{id}`,
  `GET /api/users`. Login/logout UI. Can't delete the last admin.

### 5.3 In-browser reader

- **EPUB reader.** epub.js with paginated flow (spread:none). CSS overrides
  prevent book styles from breaking column pagination. TOC sidebar from
  `book.navigation.toc`. Font size (S/M/L). Light/sepia/dark themes via
  `rendition.themes.override()`. Keyboard navigation (arrows, PageUp/Down,
  space). Per-book theme/font preferences persisted server-side.
- **PDF reader.** Browser's native PDF viewer in an iframe. `?inline=1`
  query param on the file endpoint serves with inline Content-Disposition.
- **Mobile-responsive.** Swipe left/right for page navigation (50px threshold,
  vertical swipes ignored). Nav arrows hidden on mobile (swipe replaces them).
  Compact toolbar, full-width viewport, TOC floats as overlay panel.

### 5.4 Metadata enrichment

- **Embedded subject extraction.** EPUB `<dc:subject>` and PDF keywords
  extracted as tags during scan.
- **Open Library enrichment.** Rate-limited client (1 req/sec). ISBN lookup
  → edition + works endpoints; title+author search fallback. Fills empty
  description, tags (top 15 subjects, normalized), series, publisher, cover.
  Non-destructive: never overwrites non-empty fields.
- **Tag normalization.** Canonical genre mapping (100+ entries) for verbose
  OL/EPUB subjects. Strips noise (places, curricula, metadata artifacts).
  Sidebar filters tags with <2 books to reduce clutter.
- **Admin tools.** "Rescan embedded metadata" re-extracts from all files.
  "Enrich all from Open Library" batch enrichment. "Refresh metadata" button
  on book detail page.
- **Async enrichment.** Scanner queues newly added books for background
  enrichment via a buffered channel drained by a worker goroutine.

### 5.5 Scan UX

- **fsnotify watcher.** Recursive per-root, auto-subscribes to new
  subdirectories. 2-second debounce before triggering scan. Watcher +
  periodic ticker run concurrently.
- **SSE scan events.** `GET /api/scan/{id}/events` streams job snapshots
  every 500ms. Keepalive comments every 15s. Rescan button on the home page
  uses EventSource for live progress.
- **Scanner pub-sub.** Internal broadcast of ScanJob snapshots to subscriber
  channels. Slow subscribers drop frames rather than blocking.

### 5.6 Upload

- **Browser upload.** `POST /api/books/upload` accepts multipart EPUB/PDF/
  MOBI/AZW3 files (max 100MB). Validates format, sanitizes filename, writes
  to `LIBRARY_ROOT/uploads/` with collision-safe naming. Inline processing
  (hash → metadata → upsert → cover → enrich queue) so books appear
  immediately without waiting for a scan cycle.
- **Upload dialog.** Drag-and-drop zone + file picker. File list with remove
  buttons. XHR upload with progress bar. Success results link to new book
  detail pages.

### 5.7 Duplicate detection & book management

- **Duplicate detection.** Two strategies: shared ISBN, same normalized
  title + first author. Admin-only `/admin/duplicates` page with grouped
  suspects showing cover, path, format, size, ISBN.
- **Book deletion.** Admin-only `DELETE /api/books/{id}`. Two options on
  detail page and duplicates page: "Remove from library" (soft-delete, keep
  file) and "Delete file from disk" (permanent). Confirmation dialogs.
  Frontend visibility gated by `session.can('books', 'delete')`.

### 5.8 Infrastructure

- **Docker.** Multi-stage Dockerfile: node:22-alpine (SPA build) →
  golang:1.25-alpine (static binary with embedded SPA) → alpine:3.21
  runtime (~30MB, non-root user). docker-compose.yml for deployment.
- **CI/CD.** GitHub Actions: go test + pnpm check on every push/PR. Docker
  build + push to ghcr.io on main pushes and semver tags. GHA layer caching.
- **Embed.** SvelteKit SPA compiled to `internal/webui/dist/` and embedded
  via `//go:embed`. SPA fallback for client-side routing.

## 6. Architecture

```
┌────────────────────┐        ┌──────────────────┐
│ Web UI (Svelte 5)  │        │ KOReader, Moon+  │
│  gen'd from OpenAPI│        │ (OPDS clients)   │
└──────────┬─────────┘        └────────┬─────────┘
           │                           │
           │ HTTP/JSON                 │ OPDS/XML
           ▼                           ▼
      ┌─────────────────────────────────────┐
      │         mylib server (Go)           │
      │  ┌──────────────────────────────┐   │
      │  │  Huma v2 HTTP layer          │◄──┼─── OpenAPI 3.1 spec
      │  │  (chi router, Casbin RBAC)   │   │     (auto-generated)
      │  └──────────────┬───────────────┘   │
      │  ┌──────────────┴───────────────┐   │
      │  │  services: library, scan,    │   │
      │  │  metadata, enrich, auth,     │   │
      │  │  users, progress, watcher    │   │
      │  └──────────────┬───────────────┘   │
      │  ┌──────────────┴───────────────┐   │
      │  │  storage: sqlite (modernc)   │   │
      │  │  files: local fs             │   │
      │  └──────────────────────────────┘   │
      └─────────────────────────────────────┘
                     │
                     ▼
              ┌──────────────┐
              │  ~/Books/... │  (source of truth)
              └──────────────┘
```

### 6.1 Tech Stack

| Concern           | Choice                                         |
| ----------------- | ---------------------------------------------- |
| Language          | Go 1.25+                                       |
| HTTP framework    | [Huma v2](https://huma.rocks) on chi           |
| Authorization     | Casbin v2 (embedded model + policy)            |
| Database          | SQLite via `modernc.org/sqlite` (pure Go)      |
| FTS               | SQLite FTS5                                    |
| Migrations        | `pressly/goose`                                |
| EPUB parsing      | stdlib `archive/zip` + `encoding/xml`          |
| PDF metadata      | `github.com/pdfcpu/pdfcpu`                     |
| File watching     | `fsnotify`                                     |
| Enrichment        | Open Library API                               |
| Config            | env vars (`MYLIB_` prefix)                     |
| Web frontend      | Svelte 5 (runes) + SvelteKit, adapter-static   |
| EPUB reader       | epub.js                                        |
| Client generation | `openapi-typescript` + `openapi-fetch` for TS  |
| CI/CD             | GitHub Actions → ghcr.io Docker images         |

## 7. Data Model

```
books
  id, content_hash, path, format, size_bytes, mtime,
  title, sort_title, subtitle, description,
  series_id, series_index, language, isbn, publisher,
  published_at, added_at, cover_path, metadata_source, deleted_at

authors                book_authors (book_id, author_id, role)
  id, name, sort_name

series                 book_tags (book_id, tag_id)
  id, name

tags
  id, name

users
  id, username, password_hash, role, created_at

sessions
  token, user_id, created_at, expires_at

reading_progress
  user_id, book_id, position, percent, finished,
  theme, font_size, updated_at

collections            collection_books (collection_id, book_id, position)
  id, user_id, name, created_at

scan_jobs
  id, root, started_at, finished_at, status,
  files_seen, files_added, files_updated, files_removed, error
```

Full-text index: FTS5 virtual table `books_fts` over
`(title, subtitle, authors, series, description, tags)`.
Contentless with `contentless_delete=1`.

## 8. API Surface

```
POST   /api/auth/login                   → { user }
POST   /api/auth/logout
GET    /api/auth/me                      → { user }
GET    /api/auth/permissions             → { permissions[] }

GET    /api/books                        list/filter/search (multi-tag OR)
GET    /api/books/{id}
PATCH  /api/books/{id}                   metadata overrides (books:edit)
DELETE /api/books/{id}                   soft-delete (books:delete)
GET    /api/books/{id}/file              stream original file (?inline=1)
GET    /api/books/{id}/cover             cover image (ETag+immutable)
GET    /api/books/{id}/progress          current user's progress
PUT    /api/books/{id}/progress          save progress
POST   /api/books/{id}/enrich            refresh from Open Library
POST   /api/books/upload                 multipart file upload

GET    /api/authors
GET    /api/series
GET    /api/tags                         → [{name, count}]
GET    /api/collections, POST, ...
GET    /api/progress/recent              continue-reading entries
POST   /api/progress/import             localStorage migration

POST   /api/scan                         trigger rescan (scan:trigger)
GET    /api/scan/{id}
GET    /api/scan/{id}/events             SSE progress stream

GET    /api/users                        (users:manage)
POST   /api/users
DELETE /api/users/{id}

GET    /api/admin/duplicates             (admin:access)
POST   /api/admin/rescan-metadata
POST   /api/admin/enrich-all

GET    /opds                             OPDS 1.2 root catalog
GET    /opds/recent, /opds/authors, ...

GET    /api/openapi.json, /api/docs
```

## 9. Configuration

Env vars (prefix `MYLIB_`):

| Variable                 | Default    | Description                            |
| ------------------------ | ---------- | -------------------------------------- |
| `MYLIB_LIBRARY_ROOTS`    | (required) | `:`-separated paths to scan            |
| `MYLIB_DATA_DIR`         | `./data`   | SQLite DB + cached covers              |
| `MYLIB_LISTEN`           | `:8080`    | HTTP listen address                    |
| `MYLIB_SCAN_INTERVAL`    | `10m`      | Periodic rescan cadence (`0` disables) |
| `MYLIB_LOG_LEVEL`        | `info`     | `debug` / `info` / `warn` / `error`   |
| `MYLIB_ADMIN_USER`       |            | Bootstrap admin username               |
| `MYLIB_ADMIN_PASSWORD`   |            | Bootstrap admin password               |

## 10. Resolved Decisions

- **EPUB progress format:** CFI (epub.js-native, reader-agnostic).
- **PDF progress format:** `page:N` (opaque string, frontend-owned).
- **Cover storage:** files on disk, sharded by content hash.
- **SvelteKit adapter:** `adapter-static` with `embed.FS` for single-binary.
- **TS client generation:** checked-in schema via `pnpm gen:api`.
- **Auth bootstrap:** env vars (`MYLIB_ADMIN_USER`/`PASSWORD`), created on
  startup when users table is empty.
- **Reader flow:** paginated (epub.js `flow: 'paginated'`, `spread: 'none'`)
  with CSS overrides for book style conflicts.
- **Tag filtering:** OR semantics (comma-separated on the wire).

## 11. Future Work

- Format conversion via Calibre's `ebook-convert`.
- Send-to-Kindle / send-to-device email.
- Multi-library support (separate roots with names and permissions).
- OIDC / reverse-proxy auth.
- Highlights, bookmarks, notes in the reader.
- Import from Calibre DB.
- Batch tag editing.
- Reading stats / activity dashboard.
- MOBI/AZW3 real PalmDB header parsing.
