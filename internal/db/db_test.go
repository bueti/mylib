package db

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpen_AppliesMigrations(t *testing.T) {
	dir := t.TempDir()
	conn, err := Open(dir)
	require.NoError(t, err)
	defer conn.Close()

	// Every table we expect should exist.
	tables := []string{"books", "authors", "series", "tags", "book_authors", "book_tags", "scan_jobs", "books_fts"}
	for _, tbl := range tables {
		var name string
		err := conn.QueryRow(
			`SELECT name FROM sqlite_master WHERE type IN ('table','view') AND name = ?`,
			tbl,
		).Scan(&name)
		require.NoError(t, err, "table %s missing", tbl)
		assert.Equal(t, tbl, name)
	}

	// Re-opening should be idempotent.
	conn2, err := Open(dir)
	require.NoError(t, err)
	conn2.Close()
}
