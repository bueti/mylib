package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bueti/mylib/internal/auth"
	"github.com/bueti/mylib/internal/covers"
	"github.com/bueti/mylib/internal/db"
	"github.com/bueti/mylib/internal/library"
	"github.com/bueti/mylib/internal/scanner"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// authTestServer starts an httptest server with an admin + reader user
// and returns the store so callers can seed additional data.
func authTestServer(t *testing.T) (*httptest.Server, *library.Store, *library.User, *library.User) {
	t.Helper()
	dataDir := t.TempDir()
	conn, err := db.Open(dataDir)
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	store := library.New(conn)

	ctx := context.Background()
	h1, _ := auth.HashPassword("adminpw")
	admin, err := store.CreateUser(ctx, "admin", h1, library.RoleAdmin)
	require.NoError(t, err)
	h2, _ := auth.HashPassword("readerpw")
	reader, err := store.CreateUser(ctx, "reader", h2, library.RoleReader)
	require.NoError(t, err)

	coverCache, err := covers.New(dataDir)
	require.NoError(t, err)
	sc := scanner.New(store, []string{dataDir}, coverCache)
	srv := httptest.NewServer(NewRouter(Deps{Store: store, Scanner: sc, Covers: coverCache}))
	t.Cleanup(srv.Close)
	return srv, store, admin, reader
}

// login POSTs credentials and returns the session cookie.
func login(t *testing.T, srv *httptest.Server, username, password string) *http.Cookie {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"username": username, "password": password})
	res, err := http.Post(srv.URL+"/api/auth/login", "application/json", bytes.NewReader(body))
	require.NoError(t, err)
	defer res.Body.Close()
	require.Equal(t, http.StatusOK, res.StatusCode, "login failed")
	for _, c := range res.Cookies() {
		if c.Name == SessionCookieName {
			return c
		}
	}
	t.Fatal("no session cookie in login response")
	return nil
}

func TestAuth_LoginLogoutMe(t *testing.T) {
	srv, _, _, _ := authTestServer(t)

	// Wrong password.
	body, _ := json.Marshal(map[string]string{"username": "admin", "password": "wrong"})
	res, err := http.Post(srv.URL+"/api/auth/login", "application/json", bytes.NewReader(body))
	require.NoError(t, err)
	res.Body.Close()
	assert.Equal(t, http.StatusUnauthorized, res.StatusCode)

	// /api/auth/me without cookie → 401.
	res, err = http.Get(srv.URL + "/api/auth/me")
	require.NoError(t, err)
	res.Body.Close()
	assert.Equal(t, http.StatusUnauthorized, res.StatusCode)

	// Valid login.
	cookie := login(t, srv, "admin", "adminpw")

	// /me with cookie.
	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/auth/me", nil)
	req.AddCookie(cookie)
	res, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer res.Body.Close()
	assert.Equal(t, http.StatusOK, res.StatusCode)
	var me map[string]any
	require.NoError(t, json.NewDecoder(res.Body).Decode(&me))
	assert.Equal(t, "admin", me["username"])
	assert.Equal(t, "admin", me["role"])

	// Logout.
	req, _ = http.NewRequest(http.MethodPost, srv.URL+"/api/auth/logout", nil)
	req.AddCookie(cookie)
	res, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	res.Body.Close()
	assert.Equal(t, http.StatusNoContent, res.StatusCode)

	// Cookie now invalid.
	req, _ = http.NewRequest(http.MethodGet, srv.URL+"/api/auth/me", nil)
	req.AddCookie(cookie)
	res, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	res.Body.Close()
	assert.Equal(t, http.StatusUnauthorized, res.StatusCode)
}

func TestAuth_ScanRequiresLogin(t *testing.T) {
	srv, _, _, _ := authTestServer(t)

	// Unauthenticated scan → 401.
	res, err := http.Post(srv.URL+"/api/scan", "application/json", nil)
	require.NoError(t, err)
	res.Body.Close()
	assert.Equal(t, http.StatusUnauthorized, res.StatusCode)

	// Authenticated scan → 202.
	cookie := login(t, srv, "reader", "readerpw")
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/scan", nil)
	req.AddCookie(cookie)
	res, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer res.Body.Close()
	assert.Equal(t, http.StatusAccepted, res.StatusCode)

	// Wait for the background scan to finish before teardown so the
	// test's temp dir can be removed cleanly.
	var job struct {
		ID int64 `json:"id"`
	}
	require.NoError(t, json.NewDecoder(res.Body).Decode(&job))
	waitForJob(t, srv, cookie, job.ID)
}

// waitForJob polls GET /api/scan/{id} until the scan finishes.
func waitForJob(t *testing.T, srv *httptest.Server, cookie *http.Cookie, id int64) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		req, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/scan/"+itoa(id), nil)
		req.AddCookie(cookie)
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			time.Sleep(10 * time.Millisecond)
			continue
		}
		var payload struct {
			FinishedAt *time.Time `json:"finished_at"`
		}
		_ = json.NewDecoder(r.Body).Decode(&payload)
		r.Body.Close()
		if payload.FinishedAt != nil {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("scan %d did not finish in time", id)
}

func TestAuth_AdminOnlyUsers(t *testing.T) {
	srv, _, _, _ := authTestServer(t)

	// Reader can't list users.
	readerCookie := login(t, srv, "reader", "readerpw")
	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/users", nil)
	req.AddCookie(readerCookie)
	res, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	res.Body.Close()
	assert.Equal(t, http.StatusForbidden, res.StatusCode)

	// Admin can list users.
	adminCookie := login(t, srv, "admin", "adminpw")
	req, _ = http.NewRequest(http.MethodGet, srv.URL+"/api/users", nil)
	req.AddCookie(adminCookie)
	res, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer res.Body.Close()
	assert.Equal(t, http.StatusOK, res.StatusCode)

	// Admin creates a new user.
	body, _ := json.Marshal(map[string]string{"username": "new", "password": "pw12345", "role": "reader"})
	req, _ = http.NewRequest(http.MethodPost, srv.URL+"/api/users", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(adminCookie)
	res, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	res.Body.Close()
	assert.Equal(t, http.StatusCreated, res.StatusCode)

	// Admin cannot delete themselves (would be last admin).
	var adminUser struct {
		ID int64 `json:"id"`
	}
	req, _ = http.NewRequest(http.MethodGet, srv.URL+"/api/auth/me", nil)
	req.AddCookie(adminCookie)
	res, _ = http.DefaultClient.Do(req)
	require.NoError(t, json.NewDecoder(res.Body).Decode(&adminUser))
	res.Body.Close()

	req, _ = http.NewRequest(http.MethodDelete, srv.URL+"/api/users/"+itoa(adminUser.ID), nil)
	req.AddCookie(adminCookie)
	res, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	res.Body.Close()
	assert.Equal(t, http.StatusConflict, res.StatusCode)
}
