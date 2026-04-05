package library

import (
	"context"
	"testing"
	"time"

	"github.com/bueti/mylib/internal/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	conn, err := db.Open(t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	return New(conn)
}

func TestUsers_CreateGetList(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)

	n, err := s.CountUsers(ctx)
	require.NoError(t, err)
	assert.Equal(t, 0, n)

	u, err := s.CreateUser(ctx, "alice", "hash1", RoleAdmin)
	require.NoError(t, err)
	assert.Equal(t, "alice", u.Username)
	assert.True(t, u.IsAdmin())

	got, err := s.GetUserByName(ctx, "alice")
	require.NoError(t, err)
	assert.Equal(t, u.ID, got.ID)
	assert.Equal(t, "hash1", got.PasswordHash)

	gotByID, err := s.GetUserByID(ctx, u.ID)
	require.NoError(t, err)
	assert.Equal(t, "alice", gotByID.Username)

	_, err = s.GetUserByName(ctx, "nope")
	assert.ErrorIs(t, err, ErrNotFound)

	_, err = s.CreateUser(ctx, "bob", "hash2", RoleReader)
	require.NoError(t, err)

	all, err := s.ListUsers(ctx)
	require.NoError(t, err)
	require.Len(t, all, 2)
	assert.Equal(t, "alice", all[0].Username)
	assert.Equal(t, "bob", all[1].Username)
}

func TestUsers_InvalidRole(t *testing.T) {
	_, err := newTestStore(t).CreateUser(context.Background(), "x", "h", Role("mod"))
	require.Error(t, err)
}

func TestUsers_Delete(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)
	u, err := s.CreateUser(ctx, "alice", "h", RoleReader)
	require.NoError(t, err)
	require.NoError(t, s.DeleteUser(ctx, u.ID))
	_, err = s.GetUserByID(ctx, u.ID)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestSessions_Lifecycle(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)
	u, err := s.CreateUser(ctx, "alice", "h", RoleReader)
	require.NoError(t, err)

	sess, err := s.CreateSession(ctx, "tok-abc", u.ID, time.Hour)
	require.NoError(t, err)
	assert.Equal(t, u.ID, sess.UserID)

	got, err := s.GetSessionByToken(ctx, "tok-abc")
	require.NoError(t, err)
	assert.Equal(t, u.ID, got.UserID)

	require.NoError(t, s.DeleteSession(ctx, "tok-abc"))
	_, err = s.GetSessionByToken(ctx, "tok-abc")
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestSessions_Expired(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)
	u, err := s.CreateUser(ctx, "alice", "h", RoleReader)
	require.NoError(t, err)

	// Create with negative TTL → already expired.
	_, err = s.CreateSession(ctx, "expired", u.ID, -time.Hour)
	require.NoError(t, err)

	_, err = s.GetSessionByToken(ctx, "expired")
	assert.ErrorIs(t, err, ErrNotFound)

	removed, err := s.DeleteExpiredSessions(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(1), removed)
}
