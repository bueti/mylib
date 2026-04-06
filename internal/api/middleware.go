package api

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/bueti/mylib/internal/authz"
	"github.com/bueti/mylib/internal/library"
)

// SessionCookieName is the HTTP cookie that carries the session token.
const SessionCookieName = "mylib_session"

// SessionTTL is how long a session remains valid from issuance.
const SessionTTL = 30 * 24 * time.Hour

type ctxKey int

const (
	ctxUserKey ctxKey = iota
)

// UserFromContext returns the authenticated user on the request, or
// nil if anonymous.
func UserFromContext(ctx context.Context) *library.User {
	u, _ := ctx.Value(ctxUserKey).(*library.User)
	return u
}

// OptionalAuth attaches the authenticated user to the request context
// when a valid session cookie is present, and is a no-op otherwise.
// It never rejects the request.
func OptionalAuth(store *library.Store) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if u := userFromCookie(r, store); u != nil {
				r = r.WithContext(context.WithValue(r.Context(), ctxUserKey, u))
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequireAuth rejects unauthenticated requests with 401. Handlers
// downstream can pull the user via UserFromContext.
func RequireAuth(store *library.Store) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			u := userFromCookie(r, store)
			if u == nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			r = r.WithContext(context.WithValue(r.Context(), ctxUserKey, u))
			next.ServeHTTP(w, r)
		})
	}
}

// Authorize returns middleware that checks the authenticated user's
// role against the Casbin policy for the given resource and action.
// Must be applied after RequireAuth.
func Authorize(az *authz.Authorizer, resource, action string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			u := UserFromContext(r.Context())
			if u == nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			if !az.Can(string(u.Role), resource, action) {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// userFromCookie resolves the current user from the session cookie,
// or returns nil when missing/invalid.
func userFromCookie(r *http.Request, store *library.Store) *library.User {
	c, err := r.Cookie(SessionCookieName)
	if err != nil || c.Value == "" {
		return nil
	}
	sess, err := store.GetSessionByToken(r.Context(), c.Value)
	if err != nil {
		return nil
	}
	u, err := store.GetUserByID(r.Context(), sess.UserID)
	if err != nil {
		return nil
	}
	return u
}

// SetSessionCookie writes the session cookie on the response.
func SetSessionCookie(w http.ResponseWriter, r *http.Request, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   r.TLS != nil,
		Expires:  time.Now().Add(SessionTTL),
	})
}

// ClearSessionCookie sets an expired cookie to clear the session.
func ClearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})
}

// ErrNoUser is returned from handlers when the request context doesn't
// have a user but the handler expects one. This should never happen
// after RequireAuth.
var ErrNoUser = errors.New("no user in context")
