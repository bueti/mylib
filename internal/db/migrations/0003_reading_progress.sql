-- +goose Up
-- +goose StatementBegin
CREATE TABLE reading_progress (
    user_id     INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    book_id     INTEGER NOT NULL REFERENCES books(id) ON DELETE CASCADE,
    position    TEXT NOT NULL,
    percent     REAL NOT NULL DEFAULT 0,
    finished    INTEGER NOT NULL DEFAULT 0,
    theme       TEXT,
    font_size   TEXT,
    updated_at  INTEGER NOT NULL,
    PRIMARY KEY (user_id, book_id)
);

CREATE INDEX idx_progress_user_updated ON reading_progress(user_id, updated_at DESC);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS reading_progress;
-- +goose StatementEnd
