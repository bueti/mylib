-- +goose Up
ALTER TABLE books ADD COLUMN metadata_source TEXT;

-- +goose Down
ALTER TABLE books DROP COLUMN metadata_source;
