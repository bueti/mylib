package library

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCollections_CRUD(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)
	uid, _ := seedUserAndBook(t, s, "alice")

	// Create.
	c, err := s.CreateCollection(ctx, uid, "Reading list")
	require.NoError(t, err)
	assert.Equal(t, "Reading list", c.Name)

	// Duplicate name for same user → conflict.
	_, err = s.CreateCollection(ctx, uid, "Reading list")
	require.Error(t, err)

	// List.
	all, err := s.ListCollections(ctx, uid)
	require.NoError(t, err)
	require.Len(t, all, 1)
	assert.Equal(t, "Reading list", all[0].Name)
	assert.Equal(t, 0, all[0].BookCount)

	// Rename.
	require.NoError(t, s.RenameCollection(ctx, uid, c.ID, "2026"))
	got, err := s.GetCollection(ctx, uid, c.ID)
	require.NoError(t, err)
	assert.Equal(t, "2026", got.Name)

	// Delete.
	require.NoError(t, s.DeleteCollection(ctx, uid, c.ID))
	_, err = s.GetCollection(ctx, uid, c.ID)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestCollections_UserIsolation(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)
	aliceID, _ := seedUserAndBook(t, s, "alice")
	bobID, _ := seedUserAndBook(t, s, "bob")

	c, err := s.CreateCollection(ctx, aliceID, "Private")
	require.NoError(t, err)

	// Bob cannot see or list alice's collection.
	_, err = s.GetCollection(ctx, bobID, c.ID)
	assert.ErrorIs(t, err, ErrNotFound)

	list, err := s.ListCollections(ctx, bobID)
	require.NoError(t, err)
	assert.Empty(t, list)
}

func TestCollections_RenameBelongsToOwner(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)
	aliceID, _ := seedUserAndBook(t, s, "alice")
	bobID, _ := seedUserAndBook(t, s, "bob")

	c, err := s.CreateCollection(ctx, aliceID, "Private")
	require.NoError(t, err)

	err = s.RenameCollection(ctx, bobID, c.ID, "stolen")
	assert.ErrorIs(t, err, ErrNotFound)
	// Alice's collection unchanged.
	got, err := s.GetCollection(ctx, aliceID, c.ID)
	require.NoError(t, err)
	assert.Equal(t, "Private", got.Name)
}

func TestCollections_BookMembership(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)
	uid, bid := seedUserAndBook(t, s, "alice")

	c, err := s.CreateCollection(ctx, uid, "cc")
	require.NoError(t, err)

	require.NoError(t, s.AddBookToCollection(ctx, uid, c.ID, bid))
	// Idempotent.
	require.NoError(t, s.AddBookToCollection(ctx, uid, c.ID, bid))

	list, err := s.ListCollections(ctx, uid)
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, 1, list[0].BookCount)

	require.NoError(t, s.RemoveBookFromCollection(ctx, uid, c.ID, bid))
	list, err = s.ListCollections(ctx, uid)
	require.NoError(t, err)
	assert.Equal(t, 0, list[0].BookCount)

	// Book itself remains.
	_, err = s.GetBook(ctx, bid)
	require.NoError(t, err)
}

func TestCollections_AddByNonOwner(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)
	aliceID, aliceBook := seedUserAndBook(t, s, "alice")
	bobID, _ := seedUserAndBook(t, s, "bob")

	c, err := s.CreateCollection(ctx, aliceID, "cc")
	require.NoError(t, err)

	// Bob cannot add to alice's collection.
	err = s.AddBookToCollection(ctx, bobID, c.ID, aliceBook)
	assert.ErrorIs(t, err, ErrNotFound)
}
