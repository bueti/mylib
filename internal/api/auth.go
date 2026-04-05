package api

import (
	"net/http"
	"time"

	"github.com/bueti/mylib/internal/auth"
	"github.com/bueti/mylib/internal/library"
	"github.com/go-chi/chi/v5"
)

// registerAuth wires login/logout/me as chi handlers (they set cookies
// which is awkward in Huma's typed-return model).
func registerAuth(r chi.Router, store *library.Store) {
	r.Post("/api/auth/login", func(w http.ResponseWriter, req *http.Request) {
		var body struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if err := decodeJSON(req, &body); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		u, err := store.GetUserByName(req.Context(), body.Username)
		if err != nil || auth.VerifyPassword(u.PasswordHash, body.Password) != nil {
			// Don't leak whether the user exists.
			http.Error(w, "invalid username or password", http.StatusUnauthorized)
			return
		}
		token, err := auth.NewSessionToken()
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		if _, err := store.CreateSession(req.Context(), token, u.ID, SessionTTL); err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		SetSessionCookie(w, req, token)
		writeJSON(w, http.StatusOK, meResponse(u))
	})

	r.Post("/api/auth/logout", func(w http.ResponseWriter, req *http.Request) {
		if c, err := req.Cookie(SessionCookieName); err == nil && c.Value != "" {
			_ = store.DeleteSession(req.Context(), c.Value)
		}
		ClearSessionCookie(w)
		w.WriteHeader(http.StatusNoContent)
	})

	r.Get("/api/auth/me", func(w http.ResponseWriter, req *http.Request) {
		u := userFromCookie(req, store)
		if u == nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		writeJSON(w, http.StatusOK, meResponse(u))
	})
}

// registerUsers wires admin-only user management.
func registerUsers(r chi.Router, store *library.Store) {
	r.With(RequireAuth(store), RequireAdmin).Get("/api/users", func(w http.ResponseWriter, req *http.Request) {
		users, err := store.ListUsers(req.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		out := make([]map[string]any, 0, len(users))
		for _, u := range users {
			out = append(out, meResponse(u))
		}
		writeJSON(w, http.StatusOK, map[string]any{"users": out})
	})

	r.With(RequireAuth(store), RequireAdmin).Post("/api/users", func(w http.ResponseWriter, req *http.Request) {
		var body struct {
			Username string `json:"username"`
			Password string `json:"password"`
			Role     string `json:"role"`
		}
		if err := decodeJSON(req, &body); err != nil || body.Username == "" || body.Password == "" {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		role := library.Role(body.Role)
		if role == "" {
			role = library.RoleReader
		}
		if role != library.RoleAdmin && role != library.RoleReader {
			http.Error(w, "invalid role", http.StatusBadRequest)
			return
		}
		hash, err := auth.HashPassword(body.Password)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		u, err := store.CreateUser(req.Context(), body.Username, hash, role)
		if err != nil {
			// Likely UNIQUE constraint.
			http.Error(w, "could not create user (duplicate?)", http.StatusConflict)
			return
		}
		writeJSON(w, http.StatusCreated, meResponse(u))
	})

	r.With(RequireAuth(store), RequireAdmin).Delete("/api/users/{id}", func(w http.ResponseWriter, req *http.Request) {
		id := intParam(req, "id")
		if id <= 0 {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}
		// Don't let the last admin delete themselves.
		users, err := store.ListUsers(req.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		admins := 0
		var target *library.User
		for _, u := range users {
			if u.IsAdmin() {
				admins++
			}
			if u.ID == id {
				target = u
			}
		}
		if target == nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		if target.IsAdmin() && admins <= 1 {
			http.Error(w, "cannot delete last admin", http.StatusConflict)
			return
		}
		if err := store.DeleteUser(req.Context(), id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
}

func meResponse(u *library.User) map[string]any {
	return map[string]any{
		"id":         u.ID,
		"username":   u.Username,
		"role":       string(u.Role),
		"created_at": u.CreatedAt.UTC().Format(time.RFC3339),
	}
}
