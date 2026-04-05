package library

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func seedUserAndBook(t *testing.T, s *Store, username string) (int64, int64) {
	t.Helper()
	ctx := context.Background()
	u, err := s.CreateUser(ctx, username, "h", RoleReader)
	require.NoError(t, err)
	bookID, err := s.UpsertBook(ctx, &Book{
		ContentHash: "hash-" + username, Path: "/p-" + username + ".epub", Format: "epub",
		SizeBytes: 1, Title: username + " book", SortTitle: username + " book",
	})
	require.NoError(t, err)
	return u.ID, bookID
}

func TestProgress_UpsertGet(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)
	uid, bid := seedUserAndBook(t, s, "alice")

	_, err := s.GetProgress(ctx, uid, bid)
	assert.ErrorIs(t, err, ErrNotFound)

	require.NoError(t, s.UpsertProgress(ctx, &Progress{
		UserID: uid, BookID: bid, Position: "epubcfi(/6/4[chap01]!/4/2)", Percent: 0.12,
	}))

	p, err := s.GetProgress(ctx, uid, bid)
	require.NoError(t, err)
	assert.Equal(t, "epubcfi(/6/4[chap01]!/4/2)", p.Position)
	assert.InDelta(t, 0.12, p.Percent, 0.0001)
	assert.False(t, p.Finished)

	// Update with preserved theme (COALESCE path).
	require.NoError(t, s.UpsertProgress(ctx, &Progress{
		UserID: uid, BookID: bid, Position: "epubcfi(/6/6)", Percent: 0.34, Theme: "dark",
	}))
	p, err = s.GetProgress(ctx, uid, bid)
	require.NoError(t, err)
	assert.Equal(t, "dark", p.Theme)
	// Subsequent update without theme keeps it.
	require.NoError(t, s.UpsertProgress(ctx, &Progress{
		UserID: uid, BookID: bid, Position: "epubcfi(/6/8)", Percent: 0.5,
	}))
	p, err = s.GetProgress(ctx, uid, bid)
	require.NoError(t, err)
	assert.Equal(t, "dark", p.Theme)
	assert.InDelta(t, 0.5, p.Percent, 0.0001)
}

func TestProgress_UserIsolation(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)
	aliceID, aliceBook := seedUserAndBook(t, s, "alice")
	bobID, _ := seedUserAndBook(t, s, "bob")

	require.NoError(t, s.UpsertProgress(ctx, &Progress{
		UserID: aliceID, BookID: aliceBook, Position: "a", Percent: 0.1,
	}))

	// Bob can't see alice's progress on the same book.
	_, err := s.GetProgress(ctx, bobID, aliceBook)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestProgress_Recent(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)
	uid, _ := seedUserAndBook(t, s, "alice")

	// Add three more books and mix in some progress.
	for i := 0; i < 3; i++ {
		_, err := s.UpsertBook(ctx, &Book{
			ContentHash: "h-extra-" + string(rune('a'+i)),
			Path:        "/e" + string(rune('a'+i)) + ".epub",
			Format:      "epub", SizeBytes: 1,
			Title: "Extra " + string(rune('A'+i)), SortTitle: "Extra " + string(rune('A'+i)),
		})
		require.NoError(t, err)
	}
	all, _, err := s.ListBooks(ctx, BookFilter{})
	require.NoError(t, err)
	require.Len(t, all, 4)

	// Progress on all but one — sleep between upserts so their
	// updated_at values are strictly increasing (ms granularity).
	for _, b := range all[:3] {
		require.NoError(t, s.UpsertProgress(ctx, &Progress{UserID: uid, BookID: b.ID, Position: "x", Percent: 0.1}))
		time.Sleep(2 * time.Millisecond)
	}
	// Mark one as finished — shouldn't appear in recent.
	require.NoError(t, s.UpsertProgress(ctx, &Progress{UserID: uid, BookID: all[0].ID, Position: "x", Percent: 1.0, Finished: true}))

	recent, err := s.RecentProgress(ctx, uid, 10)
	require.NoError(t, err)
	assert.Len(t, recent, 2)
	// Recent-first ordering: the last upsert was on all[2].
	assert.Equal(t, all[2].ID, recent[0].Book.ID)
}

func TestProgress_BulkImport(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)
	uid, bid := seedUserAndBook(t, s, "alice")

	// Pre-existing progress should not be overwritten.
	require.NoError(t, s.UpsertProgress(ctx, &Progress{UserID: uid, BookID: bid, Position: "server", Percent: 0.5}))

	n, err := s.BulkImportProgress(ctx, uid, []Progress{
		{BookID: bid, Position: "imported", Percent: 0.1},   // conflict → ignored
		{BookID: bid + 999, Position: "nope", Percent: 0.2}, // missing book — but insert OR IGNORE won't catch FK
	})
	// The second entry will fail or get rejected. With foreign keys on,
	// SQLite raises FK violation rather than silently inserting.
	// Accept either 0 or 1 imported depending on FK enforcement.
	require.Error(t, err)
	_ = n

	// Real use case: import against a fresh book.
	_, bid2 := seedUserAndBook(t, s, "bob")
	// bob created their own book; use that id as alice's second import target.
	n, err = s.BulkImportProgress(ctx, uid, []Progress{
		{BookID: bid2, Position: "fresh", Percent: 0.3},
	})
	require.NoError(t, err)
	assert.Equal(t, 1, n)
	p, err := s.GetProgress(ctx, uid, bid2)
	require.NoError(t, err)
	assert.Equal(t, "fresh", p.Position)

	// Original server-side progress on first book still intact.
	p, err = s.GetProgress(ctx, uid, bid)
	require.NoError(t, err)
	assert.Equal(t, "server", p.Position)
}
