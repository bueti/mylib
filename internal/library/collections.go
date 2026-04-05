package library

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

// Collection is a user-owned named grouping of books.
type Collection struct {
	ID        int64
	UserID    int64
	Name      string
	CreatedAt time.Time
	BookCount int // populated by ListCollections
}

// CreateCollection inserts a new collection for user.
func (s *Store) CreateCollection(ctx context.Context, userID int64, name string) (*Collection, error) {
	if name == "" {
		return nil, errors.New("name is required")
	}
	now := time.Now().Unix()
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO collections (user_id, name, created_at) VALUES (?, ?, ?)`,
		userID, name, now,
	)
	if err != nil {
		return nil, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	return &Collection{ID: id, UserID: userID, Name: name, CreatedAt: time.Unix(now, 0)}, nil
}

// GetCollection returns a single collection, enforcing user ownership.
func (s *Store) GetCollection(ctx context.Context, userID, id int64) (*Collection, error) {
	c := &Collection{ID: id, UserID: userID}
	var created int64
	err := s.db.QueryRowContext(ctx,
		`SELECT name, created_at FROM collections WHERE id = ? AND user_id = ?`, id, userID,
	).Scan(&c.Name, &created)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	c.CreatedAt = time.Unix(created, 0)
	return c, nil
}

// ListCollections returns all of user's collections, ordered by name.
func (s *Store) ListCollections(ctx context.Context, userID int64) ([]*Collection, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT c.id, c.name, c.created_at,
		       COALESCE((SELECT COUNT(*) FROM collection_books cb WHERE cb.collection_id = c.id), 0)
		FROM collections c
		WHERE c.user_id = ?
		ORDER BY c.name`, userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*Collection
	for rows.Next() {
		c := &Collection{UserID: userID}
		var created int64
		if err := rows.Scan(&c.ID, &c.Name, &created, &c.BookCount); err != nil {
			return nil, err
		}
		c.CreatedAt = time.Unix(created, 0)
		out = append(out, c)
	}
	return out, rows.Err()
}

// RenameCollection updates a collection name.
func (s *Store) RenameCollection(ctx context.Context, userID, id int64, name string) error {
	if name == "" {
		return errors.New("name is required")
	}
	res, err := s.db.ExecContext(ctx,
		`UPDATE collections SET name = ? WHERE id = ? AND user_id = ?`, name, id, userID,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteCollection removes a collection.
func (s *Store) DeleteCollection(ctx context.Context, userID, id int64) error {
	res, err := s.db.ExecContext(ctx,
		`DELETE FROM collections WHERE id = ? AND user_id = ?`, id, userID,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// AddBookToCollection inserts a (collection, book) link. Silently
// succeeds if the link already exists. Returns ErrNotFound if the
// collection doesn't belong to userID.
func (s *Store) AddBookToCollection(ctx context.Context, userID, collectionID, bookID int64) error {
	if _, err := s.GetCollection(ctx, userID, collectionID); err != nil {
		return err
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT OR IGNORE INTO collection_books (collection_id, book_id, position)
		 VALUES (?, ?, COALESCE((SELECT MAX(position)+1 FROM collection_books WHERE collection_id = ?), 0))`,
		collectionID, bookID, collectionID,
	)
	return err
}

// RemoveBookFromCollection removes the link.
func (s *Store) RemoveBookFromCollection(ctx context.Context, userID, collectionID, bookID int64) error {
	if _, err := s.GetCollection(ctx, userID, collectionID); err != nil {
		return err
	}
	_, err := s.db.ExecContext(ctx,
		`DELETE FROM collection_books WHERE collection_id = ? AND book_id = ?`,
		collectionID, bookID,
	)
	return err
}
