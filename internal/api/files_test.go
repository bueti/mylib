package api

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/bueti/mylib/internal/covers"
	"github.com/bueti/mylib/internal/db"
	"github.com/bueti/mylib/internal/library"
	"github.com/bueti/mylib/internal/scanner"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPI_CoverAndFileDownload(t *testing.T) {
	ctx := context.Background()
	dataDir := t.TempDir()
	libRoot := t.TempDir()
	conn, err := db.Open(dataDir)
	require.NoError(t, err)
	defer conn.Close()
	store := library.New(conn)
	coverCache, err := covers.New(dataDir)
	require.NoError(t, err)

	// Write a tiny epub with PNG cover, store directly.
	bookPath := filepath.Join(libRoot, "t.epub")
	require.NoError(t, os.WriteFile(bookPath, []byte("EPUB BYTES"), 0o644))

	coverRel, err := coverCache.Store("abcdef0123", []byte{0x89, 0x50, 0x4e, 0x47}, "image/png")
	require.NoError(t, err)

	_, err = store.UpsertBook(ctx, &library.Book{
		ContentHash: "abcdef0123", Path: bookPath, Format: "epub",
		SizeBytes: 10, Title: "T", SortTitle: "T", CoverPath: coverRel,
	})
	require.NoError(t, err)

	sc := scanner.New(store, []string{libRoot}, coverCache)
	srv := httptest.NewServer(NewRouter(Deps{Store: store, Scanner: sc, Covers: coverCache, Authz: testAuthz(t)}))
	defer srv.Close()

	books, _, err := store.ListBooks(ctx, library.BookFilter{})
	require.NoError(t, err)
	require.Len(t, books, 1)
	id := books[0].ID

	// Cover
	res, err := http.Get(srv.URL + "/api/books/" + itoa(id) + "/cover")
	require.NoError(t, err)
	defer res.Body.Close()
	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Equal(t, `"abcdef0123"`, res.Header.Get("ETag"))
	assert.Contains(t, res.Header.Get("Cache-Control"), "immutable")
	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	assert.Equal(t, []byte{0x89, 0x50, 0x4e, 0x47}, body)

	// File
	res, err = http.Get(srv.URL + "/api/books/" + itoa(id) + "/file")
	require.NoError(t, err)
	defer res.Body.Close()
	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Equal(t, "application/epub+zip", res.Header.Get("Content-Type"))
	assert.Contains(t, res.Header.Get("Content-Disposition"), ".epub")
	body, err = io.ReadAll(res.Body)
	require.NoError(t, err)
	assert.Equal(t, []byte("EPUB BYTES"), body)

	// Range request
	req, err := http.NewRequest(http.MethodGet, srv.URL+"/api/books/"+itoa(id)+"/file", nil)
	require.NoError(t, err)
	req.Header.Set("Range", "bytes=5-8")
	res, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer res.Body.Close()
	assert.Equal(t, http.StatusPartialContent, res.StatusCode)
	body, err = io.ReadAll(res.Body)
	require.NoError(t, err)
	assert.Equal(t, []byte("BYTE"), body)

	// 404 for unknown book
	res, err = http.Get(srv.URL + "/api/books/9999/cover")
	require.NoError(t, err)
	res.Body.Close()
	assert.Equal(t, http.StatusNotFound, res.StatusCode)
}
