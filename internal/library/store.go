package library

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"
)

// ErrNotFound is returned by the store when a lookup misses.
var ErrNotFound = errors.New("not found")

// Store wraps a *sql.DB with typed CRUD and search operations over the
// library schema. Methods are safe for concurrent use.
type Store struct {
	db *sql.DB
}

// New returns a Store backed by db. The caller retains ownership of db.
func New(db *sql.DB) *Store { return &Store{db: db} }

// DB exposes the underlying *sql.DB for callers that need transactions
// (e.g. the scanner).
func (s *Store) DB() *sql.DB { return s.db }

// UpsertBook inserts or updates a Book keyed by content_hash. Authors,
// series and tags are resolved (creating rows as needed) and the FTS
// index is refreshed. Returns the book's id.
func (s *Store) UpsertBook(ctx context.Context, b *Book) (int64, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	var seriesID sql.NullInt64
	if b.SeriesName != "" {
		id, err := upsertSeries(ctx, tx, b.SeriesName)
		if err != nil {
			return 0, fmt.Errorf("upsert series: %w", err)
		}
		seriesID = sql.NullInt64{Int64: id, Valid: true}
	}

	// Match an existing row: first by path (same file on disk, possibly
	// with edited content), then by content_hash (same content, file may
	// have been moved). If neither matches, insert.
	var (
		id      int64
		existed bool
	)
	err = tx.QueryRowContext(ctx,
		`SELECT id FROM books WHERE path = ?`, b.Path,
	).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		err = tx.QueryRowContext(ctx,
			`SELECT id FROM books WHERE content_hash = ?`, b.ContentHash,
		).Scan(&id)
	}
	switch {
	case errors.Is(err, sql.ErrNoRows):
		// insert
	case err != nil:
		return 0, err
	default:
		existed = true
	}

	if existed {
		_, err = tx.ExecContext(ctx, `
			UPDATE books SET
				content_hash = ?, path = ?, format = ?, size_bytes = ?, mtime = ?,
				title = ?, sort_title = ?, subtitle = ?, description = ?,
				series_id = ?, series_index = ?, language = ?, isbn = ?,
				publisher = ?, published_at = ?, cover_path = ?, deleted_at = NULL
			WHERE id = ?`,
			b.ContentHash, b.Path, b.Format, b.SizeBytes, b.MTime.Unix(),
			b.Title, b.SortTitle, nullIfEmpty(b.Subtitle), nullIfEmpty(b.Description),
			seriesID, nullFloat(b.SeriesIndex), nullIfEmpty(b.Language), nullIfEmpty(b.ISBN),
			nullIfEmpty(b.Publisher), nullIfEmpty(b.PublishedAt), nullIfEmpty(b.CoverPath),
			id,
		)
		if err != nil {
			return 0, fmt.Errorf("update book: %w", err)
		}
	} else {
		added := b.AddedAt
		if added.IsZero() {
			added = time.Now()
		}
		res, err := tx.ExecContext(ctx, `
			INSERT INTO books (
				content_hash, path, format, size_bytes, mtime,
				title, sort_title, subtitle, description,
				series_id, series_index, language, isbn,
				publisher, published_at, added_at, cover_path
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			b.ContentHash, b.Path, b.Format, b.SizeBytes, b.MTime.Unix(),
			b.Title, b.SortTitle, nullIfEmpty(b.Subtitle), nullIfEmpty(b.Description),
			seriesID, nullFloat(b.SeriesIndex), nullIfEmpty(b.Language), nullIfEmpty(b.ISBN),
			nullIfEmpty(b.Publisher), nullIfEmpty(b.PublishedAt), added.Unix(), nullIfEmpty(b.CoverPath),
		)
		if err != nil {
			return 0, fmt.Errorf("insert book: %w", err)
		}
		id, err = res.LastInsertId()
		if err != nil {
			return 0, err
		}
	}

	// Replace author links.
	if _, err := tx.ExecContext(ctx, `DELETE FROM book_authors WHERE book_id = ?`, id); err != nil {
		return 0, err
	}
	for _, a := range b.Authors {
		authorID, err := upsertAuthor(ctx, tx, a.Name)
		if err != nil {
			return 0, fmt.Errorf("upsert author: %w", err)
		}
		if _, err := tx.ExecContext(ctx,
			`INSERT OR IGNORE INTO book_authors (book_id, author_id, role) VALUES (?, ?, 'author')`,
			id, authorID,
		); err != nil {
			return 0, err
		}
	}

	// Replace tag links.
	if _, err := tx.ExecContext(ctx, `DELETE FROM book_tags WHERE book_id = ?`, id); err != nil {
		return 0, err
	}
	for _, tag := range b.Tags {
		tagID, err := upsertTag(ctx, tx, tag)
		if err != nil {
			return 0, err
		}
		if _, err := tx.ExecContext(ctx,
			`INSERT OR IGNORE INTO book_tags (book_id, tag_id) VALUES (?, ?)`, id, tagID,
		); err != nil {
			return 0, err
		}
	}

	if err := refreshFTS(ctx, tx, id); err != nil {
		return 0, fmt.Errorf("fts: %w", err)
	}
	return id, tx.Commit()
}

// SoftDelete marks a book as deleted. The row remains in the database.
func (s *Store) SoftDelete(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE books SET deleted_at = ? WHERE id = ? AND deleted_at IS NULL`,
		time.Now().Unix(), id,
	)
	if err != nil {
		return err
	}
	// Drop from FTS so it stops appearing in search.
	_, err = s.db.ExecContext(ctx, `DELETE FROM books_fts WHERE rowid = ?`, id)
	return err
}

// GetBook returns a single book by id, including its authors and tags.
func (s *Store) GetBook(ctx context.Context, id int64) (*Book, error) {
	rows, err := s.queryBooks(ctx, `WHERE b.id = ? AND b.deleted_at IS NULL`, []any{id}, "", 1, 0)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, ErrNotFound
	}
	return rows[0], nil
}

// GetBookByPath returns the (possibly soft-deleted) book stored for path.
func (s *Store) GetBookByPath(ctx context.Context, path string) (*Book, error) {
	rows, err := s.queryBooks(ctx, `WHERE b.path = ?`, []any{path}, "", 1, 0)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, ErrNotFound
	}
	return rows[0], nil
}

// ListBooks returns books matching filter. Total count of matches is
// also returned for pagination.
func (s *Store) ListBooks(ctx context.Context, f BookFilter) ([]*Book, int, error) {
	var (
		where []string
		args  []any
	)
	where = append(where, "b.deleted_at IS NULL")

	if f.Query != "" {
		// FTS5 match: rowid join.
		where = append(where, "b.id IN (SELECT rowid FROM books_fts WHERE books_fts MATCH ?)")
		args = append(args, ftsQuery(f.Query))
	}
	if f.AuthorID != nil {
		where = append(where, "b.id IN (SELECT book_id FROM book_authors WHERE author_id = ?)")
		args = append(args, *f.AuthorID)
	}
	if f.SeriesID != nil {
		where = append(where, "b.series_id = ?")
		args = append(args, *f.SeriesID)
	}
	if f.CollectionID != nil {
		where = append(where, "b.id IN (SELECT book_id FROM collection_books WHERE collection_id = ?)")
		args = append(args, *f.CollectionID)
	}
	if len(f.Tags) > 0 {
		// OR semantics: book must have ANY of the specified tags.
		placeholders := make([]string, len(f.Tags))
		for i, tag := range f.Tags {
			placeholders[i] = "?"
			args = append(args, tag)
		}
		where = append(where, "b.id IN (SELECT bt.book_id FROM book_tags bt JOIN tags t ON t.id = bt.tag_id WHERE t.name IN ("+strings.Join(placeholders, ",")+"))")
	}
	if f.Format != "" {
		where = append(where, "b.format = ?")
		args = append(args, f.Format)
	}

	whereSQL := "WHERE " + strings.Join(where, " AND ")

	orderSQL := "ORDER BY b.sort_title ASC"
	switch f.Sort {
	case "added":
		orderSQL = "ORDER BY b.added_at ASC"
	case "-added", "":
		if f.Sort == "-added" {
			orderSQL = "ORDER BY b.added_at DESC"
		}
	case "title":
		orderSQL = "ORDER BY b.sort_title ASC"
	case "-title":
		orderSQL = "ORDER BY b.sort_title DESC"
	}

	limit := f.Limit
	if limit <= 0 {
		limit = 50
	}

	// Count
	var total int
	if err := s.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM books b "+whereSQL, args...,
	).Scan(&total); err != nil {
		return nil, 0, err
	}

	books, err := s.queryBooks(ctx, whereSQL, args, orderSQL, limit, f.Offset)
	if err != nil {
		return nil, 0, err
	}
	return books, total, nil
}

// IsPathSoftDeleted reports whether the given path belongs to a
// soft-deleted book. Used by the scanner to avoid re-adding books
// that were explicitly removed by the user.
func (s *Store) IsPathSoftDeleted(ctx context.Context, path string) bool {
	var n int
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM books WHERE path = ? AND deleted_at IS NOT NULL`, path,
	).Scan(&n)
	return err == nil && n > 0
}

// ListActivePaths returns every known (not deleted) path under root,
// keyed by path → book id. Used by the scanner to detect removals.
func (s *Store) ListActivePaths(ctx context.Context, root string) (map[string]int64, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, path FROM books WHERE deleted_at IS NULL AND path LIKE ?`,
		root+string(os.PathSeparator)+"%",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[string]int64)
	for rows.Next() {
		var id int64
		var p string
		if err := rows.Scan(&id, &p); err != nil {
			return nil, err
		}
		out[p] = id
	}
	return out, rows.Err()
}

// ListAuthors returns all authors sorted by sort_name.
func (s *Store) ListAuthors(ctx context.Context) ([]Author, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, name, sort_name FROM authors ORDER BY sort_name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Author
	for rows.Next() {
		var a Author
		if err := rows.Scan(&a.ID, &a.Name, &a.SortName); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// ListSeries returns all series sorted by name.
func (s *Store) ListSeries(ctx context.Context) ([]Series, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, name FROM series ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Series
	for rows.Next() {
		var s Series
		if err := rows.Scan(&s.ID, &s.Name); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// TagCount pairs a tag name with the number of active books using it.
type TagCount struct {
	Name  string
	Count int
}

// ListTags returns all tags sorted by name.
func (s *Store) ListTags(ctx context.Context) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT name FROM tags ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

// ListTagsWithCounts returns tags with active book counts, sorted by
// count descending (most-used first).
func (s *Store) ListTagsWithCounts(ctx context.Context) ([]TagCount, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT t.name, COUNT(DISTINCT bt.book_id) AS cnt
		FROM tags t
		JOIN book_tags bt ON bt.tag_id = t.id
		JOIN books b ON b.id = bt.book_id AND b.deleted_at IS NULL
		GROUP BY t.name
		ORDER BY cnt DESC, t.name ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []TagCount
	for rows.Next() {
		var tc TagCount
		if err := rows.Scan(&tc.Name, &tc.Count); err != nil {
			return nil, err
		}
		out = append(out, tc)
	}
	return out, rows.Err()
}

// --- scan jobs ---

// CreateScanJob records the start of a scan and returns its id.
func (s *Store) CreateScanJob(ctx context.Context, root string) (int64, error) {
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO scan_jobs (root, started_at, status) VALUES (?, ?, 'running')`,
		root, time.Now().Unix(),
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// FinishScanJob writes final counts and status.
func (s *Store) FinishScanJob(ctx context.Context, id int64, j ScanJob) error {
	finished := time.Now().Unix()
	_, err := s.db.ExecContext(ctx, `
		UPDATE scan_jobs SET
			finished_at = ?, status = ?,
			files_seen = ?, files_added = ?, files_updated = ?, files_removed = ?,
			error = ?
		WHERE id = ?`,
		finished, j.Status, j.FilesSeen, j.FilesAdded, j.FilesUpdated, j.FilesRemoved,
		nullIfEmpty(j.Error), id,
	)
	return err
}

// GetScanJob returns the scan job with id.
func (s *Store) GetScanJob(ctx context.Context, id int64) (*ScanJob, error) {
	j := &ScanJob{ID: id}
	var startedAt int64
	var finishedAt sql.NullInt64
	var errStr sql.NullString
	err := s.db.QueryRowContext(ctx, `
		SELECT root, started_at, finished_at, status,
		       files_seen, files_added, files_updated, files_removed, error
		FROM scan_jobs WHERE id = ?`, id,
	).Scan(&j.Root, &startedAt, &finishedAt, &j.Status,
		&j.FilesSeen, &j.FilesAdded, &j.FilesUpdated, &j.FilesRemoved, &errStr)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	j.StartedAt = time.Unix(startedAt, 0)
	if finishedAt.Valid {
		t := time.Unix(finishedAt.Int64, 0)
		j.FinishedAt = &t
	}
	if errStr.Valid {
		j.Error = errStr.String
	}
	return j, nil
}

// --- helpers ---

// queryBooks runs the base book SELECT with an optional WHERE + ORDER BY
// + LIMIT/OFFSET, then loads authors + tags for each row.
func (s *Store) queryBooks(ctx context.Context, whereSQL string, args []any, orderSQL string, limit, offset int) ([]*Book, error) {
	base := `
		SELECT b.id, b.content_hash, b.path, b.format, b.size_bytes, b.mtime,
		       b.title, b.sort_title, COALESCE(b.subtitle,''), COALESCE(b.description,''),
		       b.series_id, COALESCE(s.name,''), b.series_index,
		       COALESCE(b.language,''), COALESCE(b.isbn,''), COALESCE(b.publisher,''),
		       COALESCE(b.published_at,''), b.added_at, COALESCE(b.cover_path,''),
		       b.deleted_at
		FROM books b
		LEFT JOIN series s ON s.id = b.series_id
		` + whereSQL + " " + orderSQL
	if limit > 0 {
		base += fmt.Sprintf(" LIMIT %d OFFSET %d", limit, offset)
	}
	rows, err := s.db.QueryContext(ctx, base, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*Book
	for rows.Next() {
		b := &Book{}
		var seriesID sql.NullInt64
		var seriesIdx sql.NullFloat64
		var mtime, added int64
		var deletedAt sql.NullInt64
		if err := rows.Scan(&b.ID, &b.ContentHash, &b.Path, &b.Format, &b.SizeBytes, &mtime,
			&b.Title, &b.SortTitle, &b.Subtitle, &b.Description,
			&seriesID, &b.SeriesName, &seriesIdx,
			&b.Language, &b.ISBN, &b.Publisher,
			&b.PublishedAt, &added, &b.CoverPath, &deletedAt,
		); err != nil {
			return nil, err
		}
		b.MTime = time.Unix(mtime, 0)
		b.AddedAt = time.Unix(added, 0)
		if seriesID.Valid {
			b.SeriesID = &seriesID.Int64
		}
		if seriesIdx.Valid {
			b.SeriesIndex = &seriesIdx.Float64
		}
		if deletedAt.Valid {
			t := time.Unix(deletedAt.Int64, 0)
			b.DeletedAt = &t
		}
		out = append(out, b)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Hydrate authors + tags in bulk.
	if len(out) > 0 {
		if err := s.loadAuthorsAndTags(ctx, out); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func (s *Store) loadAuthorsAndTags(ctx context.Context, books []*Book) error {
	byID := make(map[int64]*Book, len(books))
	ids := make([]any, 0, len(books))
	placeholders := make([]string, 0, len(books))
	for _, b := range books {
		byID[b.ID] = b
		ids = append(ids, b.ID)
		placeholders = append(placeholders, "?")
	}
	inList := "(" + strings.Join(placeholders, ",") + ")"

	// Authors
	rows, err := s.db.QueryContext(ctx, `
		SELECT ba.book_id, a.id, a.name, a.sort_name
		FROM book_authors ba JOIN authors a ON a.id = ba.author_id
		WHERE ba.book_id IN `+inList+` ORDER BY a.sort_name`, ids...)
	if err != nil {
		return err
	}
	for rows.Next() {
		var bid int64
		var a Author
		if err := rows.Scan(&bid, &a.ID, &a.Name, &a.SortName); err != nil {
			rows.Close()
			return err
		}
		if b, ok := byID[bid]; ok {
			b.Authors = append(b.Authors, a)
		}
	}
	rows.Close()

	// Tags
	rows, err = s.db.QueryContext(ctx, `
		SELECT bt.book_id, t.name FROM book_tags bt JOIN tags t ON t.id = bt.tag_id
		WHERE bt.book_id IN `+inList+` ORDER BY t.name`, ids...)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var bid int64
		var name string
		if err := rows.Scan(&bid, &name); err != nil {
			return err
		}
		if b, ok := byID[bid]; ok {
			b.Tags = append(b.Tags, name)
		}
	}
	return rows.Err()
}

// upsertAuthor looks up or inserts an author by sort_name.
func upsertAuthor(ctx context.Context, tx *sql.Tx, name string) (int64, error) {
	sortName := SortName(name)
	var id int64
	err := tx.QueryRowContext(ctx, `SELECT id FROM authors WHERE sort_name = ?`, sortName).Scan(&id)
	if err == nil {
		return id, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return 0, err
	}
	res, err := tx.ExecContext(ctx, `INSERT INTO authors (name, sort_name) VALUES (?, ?)`, name, sortName)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func upsertSeries(ctx context.Context, tx *sql.Tx, name string) (int64, error) {
	var id int64
	err := tx.QueryRowContext(ctx, `SELECT id FROM series WHERE name = ?`, name).Scan(&id)
	if err == nil {
		return id, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return 0, err
	}
	res, err := tx.ExecContext(ctx, `INSERT INTO series (name) VALUES (?)`, name)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func upsertTag(ctx context.Context, tx *sql.Tx, name string) (int64, error) {
	var id int64
	err := tx.QueryRowContext(ctx, `SELECT id FROM tags WHERE name = ?`, name).Scan(&id)
	if err == nil {
		return id, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return 0, err
	}
	res, err := tx.ExecContext(ctx, `INSERT INTO tags (name) VALUES (?)`, name)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// refreshFTS rewrites the books_fts row for book id by re-reading
// denormalised fields from the joins.
func refreshFTS(ctx context.Context, tx *sql.Tx, id int64) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM books_fts WHERE rowid = ?`, id); err != nil {
		return err
	}
	var title, subtitle, description, seriesName string
	var authors, tagNames sql.NullString
	err := tx.QueryRowContext(ctx, `
		SELECT b.title, COALESCE(b.subtitle,''), COALESCE(b.description,''),
		       COALESCE(s.name,''),
		       (SELECT GROUP_CONCAT(a.name, ' ') FROM book_authors ba JOIN authors a ON a.id = ba.author_id WHERE ba.book_id = b.id),
		       (SELECT GROUP_CONCAT(t.name, ' ') FROM book_tags bt JOIN tags t ON t.id = bt.tag_id WHERE bt.book_id = b.id)
		FROM books b LEFT JOIN series s ON s.id = b.series_id
		WHERE b.id = ?`, id,
	).Scan(&title, &subtitle, &description, &seriesName, &authors, &tagNames)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx,
		`INSERT INTO books_fts (rowid, title, subtitle, authors, series, description, tags)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		id, title, subtitle, authors.String, seriesName, description, tagNames.String,
	)
	return err
}

// ftsQuery turns a user query into an FTS5 MATCH expression. It quotes
// every term to avoid FTS syntax injection and applies a prefix wildcard
// so partial words still match ("wiz" matches "wizard").
func ftsQuery(q string) string {
	parts := strings.Fields(q)
	if len(parts) == 0 {
		return `""`
	}
	quoted := make([]string, 0, len(parts))
	for _, p := range parts {
		safe := strings.ReplaceAll(p, `"`, ``)
		if safe == "" {
			continue
		}
		quoted = append(quoted, `"`+safe+`"*`)
	}
	return strings.Join(quoted, " ")
}

// SortName produces a "Last, First" sort key from a human name. Single
// token names are returned as-is.
func SortName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	parts := strings.Fields(name)
	if len(parts) < 2 {
		return name
	}
	last := parts[len(parts)-1]
	rest := strings.Join(parts[:len(parts)-1], " ")
	return last + ", " + rest
}

// SortTitle strips a leading article ("The", "A", "An") for sort order.
func SortTitle(title string) string {
	t := strings.TrimSpace(title)
	lower := strings.ToLower(t)
	for _, prefix := range []string{"the ", "a ", "an "} {
		if strings.HasPrefix(lower, prefix) {
			return strings.TrimSpace(t[len(prefix):])
		}
	}
	return t
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func nullFloat(f *float64) any {
	if f == nil {
		return nil
	}
	return *f
}
