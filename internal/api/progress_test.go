package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/bueti/mylib/internal/library"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// seedBook inserts a book directly via the store and returns its id.
func seedBook(t *testing.T, store *library.Store, title string) int64 {
	t.Helper()
	id, err := store.UpsertBook(context.Background(), &library.Book{
		ContentHash: "hash-" + title, Path: "/p-" + title + ".epub", Format: "epub",
		SizeBytes: 1, Title: title, SortTitle: title,
	})
	require.NoError(t, err)
	return id
}

// doReq is a small helper for authenticated JSON requests.
func doReq(t *testing.T, method, url string, cookie *http.Cookie, body any) *http.Response {
	t.Helper()
	var reader *bytes.Reader
	if body != nil {
		raw, _ := json.Marshal(body)
		reader = bytes.NewReader(raw)
	} else {
		reader = bytes.NewReader(nil)
	}
	req, err := http.NewRequest(method, url, reader)
	require.NoError(t, err)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if cookie != nil {
		req.AddCookie(cookie)
	}
	res, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	return res
}

func TestProgressAPI_PutGet(t *testing.T) {
	srv, store, _, _ := authTestServer(t)
	cookie := login(t, srv, "reader", "readerpw")
	bid := seedBook(t, store, "dune")

	// No progress yet → 404.
	res := doReq(t, "GET", srv.URL+"/api/books/"+itoa(bid)+"/progress", cookie, nil)
	res.Body.Close()
	assert.Equal(t, http.StatusNotFound, res.StatusCode)

	// Save progress.
	res = doReq(t, "PUT", srv.URL+"/api/books/"+itoa(bid)+"/progress", cookie, map[string]any{
		"position": "epubcfi(/6/4)", "percent": 0.1, "finished": false,
	})
	res.Body.Close()
	assert.Equal(t, http.StatusOK, res.StatusCode)

	// Retrieve it.
	res = doReq(t, "GET", srv.URL+"/api/books/"+itoa(bid)+"/progress", cookie, nil)
	defer res.Body.Close()
	assert.Equal(t, http.StatusOK, res.StatusCode)
	var got struct {
		BookID   int64   `json:"book_id"`
		Position string  `json:"position"`
		Percent  float64 `json:"percent"`
	}
	require.NoError(t, json.NewDecoder(res.Body).Decode(&got))
	assert.Equal(t, bid, got.BookID)
	assert.Equal(t, "epubcfi(/6/4)", got.Position)
	assert.InDelta(t, 0.1, got.Percent, 0.0001)
}

func TestProgressAPI_UnauthRejected(t *testing.T) {
	srv, store, _, _ := authTestServer(t)
	bid := seedBook(t, store, "dune")

	res := doReq(t, "GET", srv.URL+"/api/books/"+itoa(bid)+"/progress", nil, nil)
	res.Body.Close()
	assert.Equal(t, http.StatusUnauthorized, res.StatusCode)

	res = doReq(t, "PUT", srv.URL+"/api/books/"+itoa(bid)+"/progress", nil,
		map[string]any{"position": "x", "percent": 0.1, "finished": false})
	res.Body.Close()
	assert.Equal(t, http.StatusUnauthorized, res.StatusCode)
}

func TestProgressAPI_UserIsolation(t *testing.T) {
	srv, store, _, _ := authTestServer(t)
	bid := seedBook(t, store, "dune")

	aliceCookie := login(t, srv, "admin", "adminpw")
	bobCookie := login(t, srv, "reader", "readerpw")

	// Alice saves.
	res := doReq(t, "PUT", srv.URL+"/api/books/"+itoa(bid)+"/progress", aliceCookie,
		map[string]any{"position": "alice-cfi", "percent": 0.5, "finished": false})
	res.Body.Close()
	require.Equal(t, http.StatusOK, res.StatusCode)

	// Bob sees no progress (404).
	res = doReq(t, "GET", srv.URL+"/api/books/"+itoa(bid)+"/progress", bobCookie, nil)
	res.Body.Close()
	assert.Equal(t, http.StatusNotFound, res.StatusCode)

	// Bob saves his own progress, independently.
	res = doReq(t, "PUT", srv.URL+"/api/books/"+itoa(bid)+"/progress", bobCookie,
		map[string]any{"position": "bob-cfi", "percent": 0.2, "finished": false})
	res.Body.Close()
	require.Equal(t, http.StatusOK, res.StatusCode)

	// Alice still sees her own position, not Bob's.
	res = doReq(t, "GET", srv.URL+"/api/books/"+itoa(bid)+"/progress", aliceCookie, nil)
	defer res.Body.Close()
	var got struct {
		Position string `json:"position"`
	}
	require.NoError(t, json.NewDecoder(res.Body).Decode(&got))
	assert.Equal(t, "alice-cfi", got.Position)
}

func TestProgressAPI_Recent(t *testing.T) {
	srv, store, _, _ := authTestServer(t)
	cookie := login(t, srv, "reader", "readerpw")
	b1 := seedBook(t, store, "a")
	b2 := seedBook(t, store, "b")

	for _, bid := range []int64{b1, b2} {
		res := doReq(t, "PUT", srv.URL+"/api/books/"+itoa(bid)+"/progress", cookie,
			map[string]any{"position": "x", "percent": 0.2, "finished": false})
		res.Body.Close()
		require.Equal(t, http.StatusOK, res.StatusCode)
	}

	res := doReq(t, "GET", srv.URL+"/api/progress/recent?limit=10", cookie, nil)
	defer res.Body.Close()
	assert.Equal(t, http.StatusOK, res.StatusCode)
	var out struct {
		Entries []struct {
			Book     struct{ ID int64 }
			Progress struct{ Position string }
		}
	}
	require.NoError(t, json.NewDecoder(res.Body).Decode(&out))
	assert.Len(t, out.Entries, 2)
}
