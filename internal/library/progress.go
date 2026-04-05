package library

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

// Progress is one user's reading progress on one book.
type Progress struct {
	UserID    int64
	BookID    int64
	Position  string // CFI for EPUB, "page:N" for PDF — opaque to the server.
	Percent   float64
	Finished  bool
	Theme     string
	FontSize  string
	UpdatedAt time.Time
}

// RecentProgressEntry pairs a book with progress info for the
// "Continue reading" row.
type RecentProgressEntry struct {
	Book     *Book
	Progress Progress
}

// GetProgress returns the user's progress on a book, or ErrNotFound.
func (s *Store) GetProgress(ctx context.Context, userID, bookID int64) (*Progress, error) {
	p := &Progress{UserID: userID, BookID: bookID}
	var updated int64
	var finished int
	var theme, fontSize sql.NullString
	err := s.db.QueryRowContext(ctx, `
		SELECT position, percent, finished, theme, font_size, updated_at
		FROM reading_progress WHERE user_id = ? AND book_id = ?`,
		userID, bookID,
	).Scan(&p.Position, &p.Percent, &finished, &theme, &fontSize, &updated)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	p.Finished = finished != 0
	p.Theme = theme.String
	p.FontSize = fontSize.String
	p.UpdatedAt = time.UnixMilli(updated)
	return p, nil
}

// UpsertProgress saves or updates the user's progress.
func (s *Store) UpsertProgress(ctx context.Context, p *Progress) error {
	now := time.Now().UnixMilli()
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO reading_progress
			(user_id, book_id, position, percent, finished, theme, font_size, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (user_id, book_id) DO UPDATE SET
			position   = excluded.position,
			percent    = excluded.percent,
			finished   = excluded.finished,
			theme      = COALESCE(excluded.theme,      reading_progress.theme),
			font_size  = COALESCE(excluded.font_size,  reading_progress.font_size),
			updated_at = excluded.updated_at`,
		p.UserID, p.BookID, p.Position, p.Percent, boolToInt(p.Finished),
		nullIfEmpty(p.Theme), nullIfEmpty(p.FontSize), now,
	)
	return err
}

// RecentProgress returns the user's most recently updated progress
// entries, joined with book data, excluding finished books.
func (s *Store) RecentProgress(ctx context.Context, userID int64, limit int) ([]*RecentProgressEntry, error) {
	if limit <= 0 || limit > 100 {
		limit = 12
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT book_id, position, percent, finished, COALESCE(theme,''), COALESCE(font_size,''), updated_at
		FROM reading_progress
		WHERE user_id = ? AND finished = 0
		ORDER BY updated_at DESC
		LIMIT ?`,
		userID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type row struct {
		p Progress
	}
	var progress []row
	var bookIDs []int64
	for rows.Next() {
		var r row
		r.p.UserID = userID
		var updated int64
		var finished int
		if err := rows.Scan(&r.p.BookID, &r.p.Position, &r.p.Percent, &finished, &r.p.Theme, &r.p.FontSize, &updated); err != nil {
			return nil, err
		}
		r.p.Finished = finished != 0
		r.p.UpdatedAt = time.UnixMilli(updated)
		progress = append(progress, r)
		bookIDs = append(bookIDs, r.p.BookID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(progress) == 0 {
		return nil, nil
	}

	// Load books in one round-trip via placeholders.
	books, err := s.booksByIDs(ctx, bookIDs)
	if err != nil {
		return nil, err
	}

	out := make([]*RecentProgressEntry, 0, len(progress))
	for _, r := range progress {
		b, ok := books[r.p.BookID]
		if !ok {
			continue // book deleted
		}
		out = append(out, &RecentProgressEntry{Book: b, Progress: r.p})
	}
	return out, nil
}

// BulkImportProgress is used by the localStorage → server migration.
// Entries that already exist on the server are left alone (client
// data is older than server data by definition once the user is
// signed in elsewhere).
func (s *Store) BulkImportProgress(ctx context.Context, userID int64, entries []Progress) (int, error) {
	if len(entries) == 0 {
		return 0, nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT OR IGNORE INTO reading_progress
			(user_id, book_id, position, percent, finished, updated_at)
		VALUES (?, ?, ?, ?, 0, ?)`)
	if err != nil {
		return 0, err
	}
	defer stmt.Close()

	imported := 0
	for _, e := range entries {
		res, err := stmt.ExecContext(ctx, userID, e.BookID, e.Position, e.Percent, time.Now().UnixMilli())
		if err != nil {
			return 0, err
		}
		if n, _ := res.RowsAffected(); n > 0 {
			imported++
		}
	}
	return imported, tx.Commit()
}

// booksByIDs fetches books by ID with authors/tags. Returned map is
// keyed by book id.
func (s *Store) booksByIDs(ctx context.Context, ids []int64) (map[int64]*Book, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	placeholders := make([]string, len(ids))
	args := make([]any, len(ids))
	for i, id := range ids {
		placeholders[i] = "?"
		args[i] = id
	}
	whereSQL := `WHERE b.id IN (` + joinStrings(placeholders, ",") + `) AND b.deleted_at IS NULL`
	books, err := s.queryBooks(ctx, whereSQL, args, "", len(ids), 0)
	if err != nil {
		return nil, err
	}
	out := make(map[int64]*Book, len(books))
	for _, b := range books {
		out[b.ID] = b
	}
	return out, nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// joinStrings is a tiny strings.Join stand-in that avoids pulling
// strings into the hot path here.
func joinStrings(xs []string, sep string) string {
	if len(xs) == 0 {
		return ""
	}
	n := len(sep) * (len(xs) - 1)
	for _, s := range xs {
		n += len(s)
	}
	b := make([]byte, 0, n)
	b = append(b, xs[0]...)
	for _, s := range xs[1:] {
		b = append(b, sep...)
		b = append(b, s...)
	}
	return string(b)
}
