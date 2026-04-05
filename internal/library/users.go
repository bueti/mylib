package library

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

// Role identifies a user's permission level.
type Role string

const (
	RoleAdmin  Role = "admin"
	RoleReader Role = "reader"
)

// User is an authenticated account.
type User struct {
	ID           int64
	Username     string
	PasswordHash string
	Role         Role
	CreatedAt    time.Time
}

// IsAdmin reports whether u has admin role.
func (u *User) IsAdmin() bool { return u != nil && u.Role == RoleAdmin }

// CreateUser inserts a new user row.
func (s *Store) CreateUser(ctx context.Context, username, passwordHash string, role Role) (*User, error) {
	if role != RoleAdmin && role != RoleReader {
		return nil, errors.New("invalid role")
	}
	now := time.Now().Unix()
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO users (username, password_hash, role, created_at) VALUES (?, ?, ?, ?)`,
		username, passwordHash, string(role), now,
	)
	if err != nil {
		return nil, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	return &User{ID: id, Username: username, PasswordHash: passwordHash, Role: role, CreatedAt: time.Unix(now, 0)}, nil
}

// GetUserByName looks up a user by username.
func (s *Store) GetUserByName(ctx context.Context, username string) (*User, error) {
	u := &User{}
	var created int64
	var role string
	err := s.db.QueryRowContext(ctx,
		`SELECT id, username, password_hash, role, created_at FROM users WHERE username = ?`, username,
	).Scan(&u.ID, &u.Username, &u.PasswordHash, &role, &created)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	u.Role = Role(role)
	u.CreatedAt = time.Unix(created, 0)
	return u, nil
}

// GetUserByID returns the user with id.
func (s *Store) GetUserByID(ctx context.Context, id int64) (*User, error) {
	u := &User{}
	var created int64
	var role string
	err := s.db.QueryRowContext(ctx,
		`SELECT id, username, password_hash, role, created_at FROM users WHERE id = ?`, id,
	).Scan(&u.ID, &u.Username, &u.PasswordHash, &role, &created)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	u.Role = Role(role)
	u.CreatedAt = time.Unix(created, 0)
	return u, nil
}

// ListUsers returns all users sorted by username.
func (s *Store) ListUsers(ctx context.Context) ([]*User, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, username, password_hash, role, created_at FROM users ORDER BY username`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*User
	for rows.Next() {
		u := &User{}
		var created int64
		var role string
		if err := rows.Scan(&u.ID, &u.Username, &u.PasswordHash, &role, &created); err != nil {
			return nil, err
		}
		u.Role = Role(role)
		u.CreatedAt = time.Unix(created, 0)
		out = append(out, u)
	}
	return out, rows.Err()
}

// CountUsers returns the total number of users.
func (s *Store) CountUsers(ctx context.Context) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&n)
	return n, err
}

// DeleteUser removes a user and cascades to sessions/progress/collections.
func (s *Store) DeleteUser(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM users WHERE id = ?`, id)
	return err
}
