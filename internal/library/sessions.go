package library

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

// Session is a server-side auth session. The Token is the value
// stored in the client's cookie.
type Session struct {
	Token     string
	UserID    int64
	CreatedAt time.Time
	ExpiresAt time.Time
}

// CreateSession inserts a new session row.
func (s *Store) CreateSession(ctx context.Context, token string, userID int64, ttl time.Duration) (*Session, error) {
	now := time.Now()
	expires := now.Add(ttl)
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO sessions (token, user_id, created_at, expires_at) VALUES (?, ?, ?, ?)`,
		token, userID, now.Unix(), expires.Unix(),
	)
	if err != nil {
		return nil, err
	}
	return &Session{Token: token, UserID: userID, CreatedAt: now, ExpiresAt: expires}, nil
}

// GetSessionByToken fetches a non-expired session.
func (s *Store) GetSessionByToken(ctx context.Context, token string) (*Session, error) {
	var sess Session
	var created, expires int64
	err := s.db.QueryRowContext(ctx,
		`SELECT token, user_id, created_at, expires_at FROM sessions WHERE token = ? AND expires_at > ?`,
		token, time.Now().Unix(),
	).Scan(&sess.Token, &sess.UserID, &created, &expires)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	sess.CreatedAt = time.Unix(created, 0)
	sess.ExpiresAt = time.Unix(expires, 0)
	return &sess, nil
}

// DeleteSession removes a session by token.
func (s *Store) DeleteSession(ctx context.Context, token string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE token = ?`, token)
	return err
}

// DeleteExpiredSessions removes all sessions past their expiry.
func (s *Store) DeleteExpiredSessions(ctx context.Context) (int64, error) {
	res, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE expires_at <= ?`, time.Now().Unix())
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
