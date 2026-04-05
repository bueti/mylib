# mylib — Personal Ebook & PDF Library

## 1. Overview

A self-hosted library manager for ebooks (EPUB, MOBI, AZW3) and PDFs, in the
spirit of [Audiobookshelf](https://www.audiobookshelf.org/) but focused on
reading formats rather than audio. Users point the server at one or more
directories on disk; the server scans, extracts metadata, and exposes a
browsable, searchable collection via a typed HTTP API with generated clients.

## 2. Goals

- **Self-hosted, single binary.** Runs locally or on a home server with minimal
  configuration. SQLite by default; no external services required.
- **Non-destructive.** Source files on disk are the source of truth. The server
  never moves, renames, or rewrites files unless explicitly told to.
- **Typed API first.** Every endpoint is described in OpenAPI, generated
  automatically by Huma. Clients (TypeScript web UI, Go CLI, others) are
  generated from the spec — no hand-written client code.
- **Fast library operations.** Scans of 10k+ files complete in seconds on
  re-scan; full-text metadata search is sub-100ms.
- **Multi-user with progress tracking.** Each user has their own reading
  progress, bookmarks, and collections over a shared library.

## 3. Non-Goals (for v1)

- Audiobook support (that's what Audiobookshelf is for).
- In-browser reading UI. v1 ships download + OPDS; a reader can come later.
- Social features (sharing, ratings, recommendations).
- Cloud sync of user libraries across instances.
- Mobile apps. Third-party OPDS clients (KOReader, Moon+ Reader) cover this.
- DRM handling. Users are responsible for their own files.

## 4. Users & Use Cases

**Primary user:** a technical home-server operator with a personal ebook
collection spread across folders, wanting a unified catalog.

**Use cases:**

1. Point the server at `~/Books` and browse by author/series/tag in a web UI.
2. Search "Le Guin" and download an EPUB to a Kobo over the local network.
3. Open a PDF in a reader, close it, reopen later and resume on the same page
   from a different device.
4. Subscribe to the library's OPDS feed from KOReader on an e-ink device.
5. Tag a subset of books as "2026 reading list" and filter by that tag.

## 5. Key Features

### 5.1 MVP (v0.1)

- **Library scanning:** recursive directory watch; detects new, modified, and
  deleted files. Incremental rescans using mtime + size + content hash.
- **Metadata extraction:**
  - EPUB: parse OPF for title, authors, series, publisher, language, ISBN,
    description, cover image.
  - PDF: extract embedded XMP/Info metadata; fall back to filename heuristics.
  - MOBI/AZW3: parse PalmDB headers for basic metadata.
- **Metadata override:** per-book user edits stored alongside extracted data;
  user edits always win.
- **Browse & search:** by author, series, tag, format, date added, date
  published. Full-text search over title/author/description (SQLite FTS5).
- **Collections & tags:** user-defined groupings; a book can belong to many.
- **Reading progress:** per-user, per-book current position (page or CFI) plus
  last-read timestamp and a finished flag.
- **Download:** direct file download and OPDS 1.2 catalog feed.
- **Auth:** local username/password with bcrypt; sessions via signed cookies
  or bearer tokens. Admin vs. reader roles.
- **Covers:** extracted cover thumbnails cached on disk; served with
  long-lived ETags.

### 5.2 v0.2 — "Multi-user & resume reading"

The core gap after v0.1 is that reading progress lives in localStorage
and there are no user accounts. v0.2 closes both in one coherent release
so progress, collections, and favorites are meaningful across devices.

**Must-ship**
- **Local auth.** Username + bcrypt password, signed-cookie sessions,
  admin vs. reader roles, first-run admin bootstrap. All write and
  per-user endpoints require a session.
- **Server-side reading progress.** Per-user, per-book: position
  (CFI for EPUB, page for PDF), percent, updated_at, finished flag.
  Reader saves debounced every ~5s and on chapter change; resumes on
  open. Migrates existing localStorage CFIs on first login.
  - `GET /api/books/{id}/progress`, `PUT /api/books/{id}/progress`.
  - `GET /api/progress/recent` — user's recently-read for a "Continue
    reading" row on the home page.
- **Collections.** Per-user named groupings (e.g. "2026 reading list").
  A book can belong to many. Add/remove from the detail page.
  - `GET/POST /api/collections`, `POST /api/collections/{id}/books/{book_id}`.
- **Scan UX.** `fsnotify` watcher picks up new files instantly; SSE
  endpoint `GET /api/scan/{id}/events` streams live counts so the
  Rescan button shows real progress instead of polling.

**Should-ship**
- **Duplicate detection.** Surface exact-match (same content_hash) and
  probable-match (same ISBN or near-duplicate fuzzy title) in an admin
  view so users can manually resolve.
- **Reader polish.** EPUB reader gains a TOC sidebar, font-size +
  theme (light/sepia/dark), and remembers per-book preferences.

**Deferred to v0.3+**
- External metadata providers (Google Books / Open Library) with a
  per-book "refresh metadata" action.
- Format conversion via Calibre's `ebook-convert`.
- Send-to-Kindle / send-to-device email.
- Multi-library support (separate roots with names and permissions).
- OIDC / reverse-proxy auth.
- Highlights, bookmarks, notes in the reader.
- PDF first-page thumbnail generation for missing covers.

### 5.3 v0.2 success criteria

- A fresh install completes first-run setup (create admin → log in →
  point at a library → browse) in under 2 minutes.
- Two users on the same instance see independent progress, collections,
  and "Continue reading" rows.
- Opening a book on device A, reading to chapter 3, then opening it on
  device B resumes at chapter 3 within one second of page load.
- Dropping a new EPUB into the library root makes it appear in the UI
  within 5 seconds (fsnotify-triggered scan).
- Existing v0.1 localStorage progress is migrated on first login, not
  lost.

### 5.4 v0.2 non-goals

- Sharing reading lists or progress between users.
- Sync with external services (Goodreads, Storygraph, Calibre server).
- Mobile app.
- Per-collection ACLs — collections are owned by a single user.
- Migrating to Postgres. SQLite remains the only supported store.

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
      │  │  (chi router)                │   │     (auto-generated)
      │  └──────────────┬───────────────┘   │
      │  ┌──────────────┴───────────────┐   │
      │  │  services: library, scan,    │   │
      │  │  metadata, users, progress   │   │
      │  └──────────────┬───────────────┘   │
      │  ┌──────────────┴───────────────┐   │
      │  │  storage: sqlite (modernc)   │   │
      │  │  files: local fs             │   │
      │  └──────────────────────────────┘   │
      └─────────────────────────────────────┘
                     │
                     ▼
              ┌──────────────┐
              │  ~/Books/... │  (source of truth; read-only by default)
              └──────────────┘
```

### 6.1 Tech Stack

| Concern           | Choice                                         |
| ----------------- | ---------------------------------------------- |
| Language          | Go 1.26+                                       |
| HTTP framework    | [Huma v2](https://huma.rocks) on chi           |
| Database          | SQLite via `modernc.org/sqlite` (pure Go)      |
| FTS               | SQLite FTS5                                    |
| Migrations        | `pressly/goose`                                |
| EPUB parsing      | `github.com/taylorskalyo/goreader/epub` or own |
| PDF metadata      | `github.com/pdfcpu/pdfcpu`                     |
| File watching     | `fsnotify`                                     |
| Config            | env vars + optional `config.yaml`              |
| Web frontend      | Svelte 5 (runes) + SvelteKit, Vite, TypeScript |
| Client generation | `oapi-codegen` for Go,                         |
|                   | `openapi-typescript` + `openapi-fetch` for TS  |

### 6.2 Why Huma

- OpenAPI 3.1 spec emitted directly from typed Go handler signatures — the
  spec cannot drift from the implementation.
- Input/output structs give us request validation, response shapes, and
  client-generation input from one definition.
- First-class support for streaming (needed for file downloads) and SSE
  (needed for scan-progress updates).

## 7. Data Model (sketch)

```
books
  id, content_hash, path, format, size_bytes, mtime,
  title, sort_title, subtitle, description,
  series_id, series_index, language, isbn, publisher,
  published_at, added_at, cover_path

authors                book_authors (book_id, author_id, role)
  id, name, sort_name

series                 book_tags (book_id, tag_id)
  id, name

tags
  id, name

users
  id, username, password_hash, role, created_at

reading_progress
  user_id, book_id, position, percent, finished, updated_at

collections            collection_books (collection_id, book_id, position)
  id, user_id, name

scan_jobs
  id, root, started_at, finished_at, status,
  files_seen, files_added, files_updated, files_removed
```

Full-text index is a virtual FTS5 table over
`(title, subtitle, authors, series, description, tags)`.

## 8. API Surface (selected)

```
POST   /auth/login                      → { token }
POST   /auth/logout
GET    /auth/me

GET    /books                           list/filter/search
GET    /books/{id}
PATCH  /books/{id}                      user metadata overrides
GET    /books/{id}/file                 stream original file
GET    /books/{id}/cover                cover image
GET    /books/{id}/progress             current user's progress
PUT    /books/{id}/progress

GET    /authors, /series, /tags
GET    /collections, POST /collections, ...

POST   /scan                            trigger rescan
GET    /scan/{id}                       status
GET    /scan/{id}/events                SSE progress stream

GET    /opds                            OPDS 1.2 root catalog
GET    /opds/recent, /opds/authors, ... browseable subfeeds

GET    /openapi.json, /docs
```

All JSON endpoints share a typed error envelope (RFC 7807 problem+json,
which Huma produces by default).

## 9. Configuration

Env vars (prefix `MYLIB_`):

```
MYLIB_LIBRARY_ROOTS=/srv/books:/srv/pdfs   # colon-separated
MYLIB_DATA_DIR=/var/lib/mylib              # db + covers + index
MYLIB_LISTEN=:8080
MYLIB_SESSION_SECRET=...
MYLIB_SCAN_INTERVAL=10m                    # 0 disables periodic scans
MYLIB_LOG_LEVEL=info
```

## 10. Milestones

1. **Skeleton & scan:** Huma server, SQLite migrations, scanner that
   populates `books` from disk with EPUB + PDF metadata extraction.
2. **API & OpenAPI:** full CRUD for books/authors/series/tags; filters &
   FTS search; generated TS + Go clients wired up.
3. **Auth & progress:** users, sessions, per-user reading progress,
   collections.
4. **Serving:** file downloads, cover serving with ETags, OPDS feed.
5. **Scan UX:** fsnotify watcher, SSE progress stream, admin endpoints.
6. **Web UI:** minimal Svelte 5 (SvelteKit) app consuming the generated
   TypeScript client — browse, search, download. Uses runes (`$state`,
   `$derived`, `$effect`) for reactivity; no Svelte stores for new code.

## 11. Open Questions

- **Progress granularity for PDFs:** page number is easy; for EPUBs, do we
  store CFI (reader-agnostic but complex) or a simple spine-index + offset?
- **Cover storage:** inline in SQLite blob vs. files on disk. Leaning
  files-on-disk for HTTP caching simplicity.
- **Author de-duplication:** "Ursula K. Le Guin" vs. "Ursula Le Guin" vs.
  "Le Guin, Ursula K." — do we attempt fuzzy merges, or leave to manual
  admin actions?
- **Write access to the library root:** keep strictly read-only in v1, or
  allow a "organize on import" mode that moves files into
  `Author/Series/Title.ext`? (Post-MVP, opt-in.)
- **Client generation in CI:** who owns the generated code — checked in,
  or generated on build? Leaning checked-in for TS (so web devs can work
  without running the Go server) and generated-on-build for internal Go.
- **SvelteKit adapter:** `adapter-static` (ship the SPA as assets served
  by the Go binary via `embed.FS`) vs. `adapter-node` (separate process,
  SSR available). Leaning `adapter-static` for single-binary deploys.
