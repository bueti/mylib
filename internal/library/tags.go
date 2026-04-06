package library

import "context"

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
