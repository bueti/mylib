-- +goose Up
-- +goose StatementBegin
CREATE TABLE collections (
    id         INTEGER PRIMARY KEY,
    user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    UNIQUE (user_id, name)
);

CREATE TABLE collection_books (
    collection_id INTEGER NOT NULL REFERENCES collections(id) ON DELETE CASCADE,
    book_id       INTEGER NOT NULL REFERENCES books(id) ON DELETE CASCADE,
    position      INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (collection_id, book_id)
);

CREATE INDEX idx_collections_user       ON collections(user_id);
CREATE INDEX idx_collection_books_book  ON collection_books(book_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS collection_books;
DROP TABLE IF EXISTS collections;
-- +goose StatementEnd
