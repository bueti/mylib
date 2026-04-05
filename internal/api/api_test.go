package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bueti/mylib/internal/covers"
	"github.com/bueti/mylib/internal/db"
	"github.com/bueti/mylib/internal/library"
	"github.com/bueti/mylib/internal/scanner"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPI_ListAndGetBook(t *testing.T) {
	ctx := context.Background()
	dataDir := t.TempDir()
	conn, err := db.Open(dataDir)
	require.NoError(t, err)
	defer conn.Close()
	store := library.New(conn)

	// Seed two books directly via the store.
	idx := 1.0
	seriesName := "Earthsea"
	for _, b := range []*library.Book{
		{
			ContentHash: "hash-a", Path: "/a.epub", Format: "epub", SizeBytes: 100,
			Title: "A Wizard of Earthsea", SortTitle: "Wizard of Earthsea",
			SeriesName: seriesName, SeriesIndex: &idx,
			Authors: []library.Author{{Name: "Ursula Le Guin", SortName: "Le Guin, Ursula"}},
		},
		{
			ContentHash: "hash-b", Path: "/b.epub", Format: "epub", SizeBytes: 200,
			Title: "Dune", SortTitle: "Dune",
			Authors: []library.Author{{Name: "Frank Herbert", SortName: "Herbert, Frank"}},
		},
	} {
		_, err := store.UpsertBook(ctx, b)
		require.NoError(t, err)
	}

	coverCache, err := covers.New(dataDir)
	require.NoError(t, err)
	sc := scanner.New(store, []string{dataDir}, coverCache)

	handler := NewRouter(Deps{Store: store, Scanner: sc, Covers: coverCache})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	// GET /api/books
	res, err := http.Get(srv.URL + "/api/books")
	require.NoError(t, err)
	defer res.Body.Close()
	assert.Equal(t, http.StatusOK, res.StatusCode)

	var list ListBooksOutput
	require.NoError(t, json.NewDecoder(res.Body).Decode(&list.Body))
	assert.Equal(t, 2, list.Body.Total)
	require.Len(t, list.Body.Books, 2)

	// Search
	res, err = http.Get(srv.URL + "/api/books?q=dune")
	require.NoError(t, err)
	defer res.Body.Close()
	require.NoError(t, json.NewDecoder(res.Body).Decode(&list.Body))
	assert.Equal(t, 1, list.Body.Total)
	require.Len(t, list.Body.Books, 1)
	assert.Equal(t, "Dune", list.Body.Books[0].Title)

	// GET /api/books/{id} with a valid id
	firstID := list.Body.Books[0].ID
	res, err = http.Get(srv.URL + "/api/books/" + itoa(firstID))
	require.NoError(t, err)
	defer res.Body.Close()
	assert.Equal(t, http.StatusOK, res.StatusCode)

	// 404 on unknown id
	res, err = http.Get(srv.URL + "/api/books/99999")
	require.NoError(t, err)
	defer res.Body.Close()
	assert.Equal(t, http.StatusNotFound, res.StatusCode)

	// OpenAPI spec renders
	res, err = http.Get(srv.URL + "/api/openapi.json")
	require.NoError(t, err)
	defer res.Body.Close()
	assert.Equal(t, http.StatusOK, res.StatusCode)
}

func TestAPI_Taxonomy(t *testing.T) {
	ctx := context.Background()
	dataDir := t.TempDir()
	conn, err := db.Open(dataDir)
	require.NoError(t, err)
	defer conn.Close()
	store := library.New(conn)

	_, err = store.UpsertBook(ctx, &library.Book{
		ContentHash: "h", Path: "/x.epub", Format: "epub", SizeBytes: 1,
		Title: "X", SortTitle: "X", SeriesName: "Z",
		Authors: []library.Author{{Name: "Alice", SortName: "Alice"}},
		Tags:    []string{"scifi"},
	})
	require.NoError(t, err)

	coverCache, err := covers.New(dataDir)
	require.NoError(t, err)
	sc := scanner.New(store, []string{dataDir}, coverCache)
	srv := httptest.NewServer(NewRouter(Deps{Store: store, Scanner: sc, Covers: coverCache}))
	defer srv.Close()

	for _, path := range []string{"/api/authors", "/api/series", "/api/tags"} {
		res, err := http.Get(srv.URL + path)
		require.NoError(t, err, path)
		res.Body.Close()
		assert.Equal(t, http.StatusOK, res.StatusCode, path)
	}
}

func itoa(i int64) string {
	if i == 0 {
		return "0"
	}
	neg := i < 0
	if neg {
		i = -i
	}
	buf := make([]byte, 0, 20)
	for i > 0 {
		buf = append([]byte{byte('0' + i%10)}, buf...)
		i /= 10
	}
	if neg {
		buf = append([]byte("-"), buf...)
	}
	return string(buf)
}
