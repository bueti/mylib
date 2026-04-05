package opds

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bueti/mylib/internal/db"
	"github.com/bueti/mylib/internal/library"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOPDS_RootAndFeeds(t *testing.T) {
	ctx := context.Background()
	conn, err := db.Open(t.TempDir())
	require.NoError(t, err)
	defer conn.Close()
	store := library.New(conn)

	_, err = store.UpsertBook(ctx, &library.Book{
		ContentHash: "h1", Path: "/a.epub", Format: "epub", SizeBytes: 1,
		Title: "Book One", SortTitle: "Book One",
		Authors: []library.Author{{Name: "Alice Smith", SortName: "Smith, Alice"}},
	})
	require.NoError(t, err)

	r := chi.NewRouter()
	Mount(r, &Handler{Store: store})
	srv := httptest.NewServer(r)
	defer srv.Close()

	cases := []struct {
		path           string
		wantMIMEPrefix string
		wantContains   []string
	}{
		{"/opds", "application/atom+xml;profile=opds-catalog;kind=navigation",
			[]string{"urn:mylib:catalog", "Recently Added", "Authors"}},
		{"/opds/recent", "application/atom+xml;profile=opds-catalog;kind=acquisition",
			[]string{"Book One", "opds-spec.org/acquisition", "/api/books/"}},
		{"/opds/authors", "application/atom+xml;profile=opds-catalog;kind=navigation",
			[]string{"Alice Smith"}},
		{"/opds/search?q=book", "application/atom+xml;profile=opds-catalog;kind=acquisition",
			[]string{"Book One"}},
	}

	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			res, err := http.Get(srv.URL + tc.path)
			require.NoError(t, err)
			defer res.Body.Close()
			assert.Equal(t, http.StatusOK, res.StatusCode)
			assert.True(t, strings.HasPrefix(res.Header.Get("Content-Type"), tc.wantMIMEPrefix),
				"Content-Type: %s", res.Header.Get("Content-Type"))
			body, err := io.ReadAll(res.Body)
			require.NoError(t, err)
			for _, want := range tc.wantContains {
				assert.Contains(t, string(body), want)
			}
		})
	}
}
