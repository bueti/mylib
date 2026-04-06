package library

import (
	"context"
	"strings"
)

// RenormalizeTags re-processes all book tags through a normalizer
// function. Tags that normalize to empty or duplicate are removed.
// Returns the number of books updated.
func (s *Store) RenormalizeTags(ctx context.Context, normalize func([]string) []string) (int, error) {
	books, _, err := s.ListBooks(ctx, BookFilter{Limit: 50_000})
	if err != nil {
		return 0, err
	}
	updated := 0
	for _, b := range books {
		if len(b.Tags) == 0 {
			continue
		}
		cleaned := normalize(b.Tags)
		if tagsEqual(b.Tags, cleaned) {
			continue
		}
		b.Tags = cleaned
		if _, err := s.UpsertBook(ctx, b); err != nil {
			return updated, err
		}
		updated++
	}
	return updated, nil
}

// CleanOrphanTags removes tags from the tags table that are no longer
// referenced by any book. Returns the number removed.
func (s *Store) CleanOrphanTags(ctx context.Context) (int64, error) {
	res, err := s.db.ExecContext(ctx,
		`DELETE FROM tags WHERE id NOT IN (SELECT DISTINCT tag_id FROM book_tags)`)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// ApplyTagMerges replaces tags across all books according to a merge
// map (old name → canonical name). Returns the number of books updated.
func (s *Store) ApplyTagMerges(ctx context.Context, merges map[string]string) (int, error) {
	if len(merges) == 0 {
		return 0, nil
	}
	books, _, err := s.ListBooks(ctx, BookFilter{Limit: 50_000})
	if err != nil {
		return 0, err
	}
	updated := 0
	for _, b := range books {
		if len(b.Tags) == 0 {
			continue
		}
		changed := false
		seen := make(map[string]struct{})
		var newTags []string
		for _, t := range b.Tags {
			if canon, ok := merges[t]; ok {
				t = canon
				changed = true
			}
			lower := strings.ToLower(t)
			if _, dup := seen[lower]; dup {
				changed = true
				continue
			}
			seen[lower] = struct{}{}
			newTags = append(newTags, t)
		}
		if !changed {
			continue
		}
		b.Tags = newTags
		if _, err := s.UpsertBook(ctx, b); err != nil {
			return updated, err
		}
		updated++
	}
	return updated, nil
}

func tagsEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
