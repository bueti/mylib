-- +goose Up
-- +goose StatementBegin
CREATE TABLE authors (
    id        INTEGER PRIMARY KEY,
    name      TEXT NOT NULL,
    sort_name TEXT NOT NULL UNIQUE
);

CREATE TABLE series (
    id   INTEGER PRIMARY KEY,
    name TEXT NOT NULL UNIQUE
);

CREATE TABLE tags (
    id   INTEGER PRIMARY KEY,
    name TEXT NOT NULL UNIQUE
);

CREATE TABLE books (
    id            INTEGER PRIMARY KEY,
    content_hash  TEXT NOT NULL UNIQUE,
    path          TEXT NOT NULL,
    format        TEXT NOT NULL,
    size_bytes    INTEGER NOT NULL,
    mtime         INTEGER NOT NULL,
    title         TEXT NOT NULL,
    sort_title    TEXT NOT NULL,
    subtitle      TEXT,
    description   TEXT,
    series_id     INTEGER REFERENCES series(id) ON DELETE SET NULL,
    series_index  REAL,
    language      TEXT,
    isbn          TEXT,
    publisher     TEXT,
    published_at  TEXT,
    added_at      INTEGER NOT NULL,
    cover_path    TEXT,
    deleted_at    INTEGER
);

CREATE INDEX idx_books_path        ON books(path);
CREATE INDEX idx_books_sort_title  ON books(sort_title);
CREATE INDEX idx_books_added_at    ON books(added_at);
CREATE INDEX idx_books_series      ON books(series_id, series_index);
CREATE INDEX idx_books_deleted_at  ON books(deleted_at);

CREATE TABLE book_authors (
    book_id   INTEGER NOT NULL REFERENCES books(id)   ON DELETE CASCADE,
    author_id INTEGER NOT NULL REFERENCES authors(id) ON DELETE CASCADE,
    role      TEXT NOT NULL DEFAULT 'author',
    PRIMARY KEY (book_id, author_id, role)
);

CREATE INDEX idx_book_authors_author ON book_authors(author_id);

CREATE TABLE book_tags (
    book_id INTEGER NOT NULL REFERENCES books(id) ON DELETE CASCADE,
    tag_id  INTEGER NOT NULL REFERENCES tags(id)  ON DELETE CASCADE,
    PRIMARY KEY (book_id, tag_id)
);

CREATE INDEX idx_book_tags_tag ON book_tags(tag_id);

CREATE TABLE scan_jobs (
    id             INTEGER PRIMARY KEY,
    root           TEXT NOT NULL,
    started_at     INTEGER NOT NULL,
    finished_at    INTEGER,
    status         TEXT NOT NULL,
    files_seen     INTEGER NOT NULL DEFAULT 0,
    files_added    INTEGER NOT NULL DEFAULT 0,
    files_updated  INTEGER NOT NULL DEFAULT 0,
    files_removed  INTEGER NOT NULL DEFAULT 0,
    error          TEXT
);

-- FTS5 index over searchable book fields.
-- Contentless external-content table: we manage inserts/updates/deletes via
-- triggers on books so the index stays in sync without duplicating data.
CREATE VIRTUAL TABLE books_fts USING fts5(
    title, subtitle, authors, series, description, tags,
    content='', tokenize='unicode61'
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS books_fts;
DROP TABLE IF EXISTS scan_jobs;
DROP TABLE IF EXISTS book_tags;
DROP TABLE IF EXISTS book_authors;
DROP TABLE IF EXISTS books;
DROP TABLE IF EXISTS tags;
DROP TABLE IF EXISTS series;
DROP TABLE IF EXISTS authors;
-- +goose StatementEnd
