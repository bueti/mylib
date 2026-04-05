package library

import (
	"context"
	"regexp"
	"strings"
)

// DuplicateGroup is a set of books that look like duplicates of each
// other, grouped by a shared key (ISBN or normalized title+author).
type DuplicateGroup struct {
	Reason string  // "isbn" or "title"
	Key    string  // the shared ISBN or normalized title
	Books  []*Book // two or more books
}

// FindDuplicates returns suspected-duplicate groups. Two strategies
// are run and their results concatenated:
//
//  1. Books sharing an ISBN (admin should pick which to keep).
//  2. Books whose normalized title + first-author sort-name match.
//
// Each book appears in at most one group per strategy.
func (s *Store) FindDuplicates(ctx context.Context) ([]*DuplicateGroup, error) {
	// Load all active books + authors.
	books, _, err := s.ListBooks(ctx, BookFilter{Limit: 10_000, Sort: "title"})
	if err != nil {
		return nil, err
	}

	var groups []*DuplicateGroup

	// Strategy 1: ISBN.
	byISBN := make(map[string][]*Book)
	for _, b := range books {
		isbn := normalizeISBN(b.ISBN)
		if isbn == "" {
			continue
		}
		byISBN[isbn] = append(byISBN[isbn], b)
	}
	for isbn, bs := range byISBN {
		if len(bs) > 1 {
			groups = append(groups, &DuplicateGroup{Reason: "isbn", Key: isbn, Books: bs})
		}
	}

	// Strategy 2: normalized title + first author.
	byTitle := make(map[string][]*Book)
	for _, b := range books {
		key := dupeKey(b)
		if key == "" {
			continue
		}
		byTitle[key] = append(byTitle[key], b)
	}
	for key, bs := range byTitle {
		if len(bs) > 1 {
			groups = append(groups, &DuplicateGroup{Reason: "title", Key: key, Books: bs})
		}
	}
	return groups, nil
}

// dupeKey returns a normalized "title|author" fingerprint for a book,
// or "" when we don't have enough info.
func dupeKey(b *Book) string {
	title := normalizeTitle(b.Title)
	if title == "" || len(b.Authors) == 0 {
		return ""
	}
	return title + "|" + normalizeTitle(b.Authors[0].SortName)
}

var nonAlnum = regexp.MustCompile(`[^a-z0-9]+`)

func normalizeTitle(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = nonAlnum.ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}

func normalizeISBN(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	b := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c >= '0' && c <= '9') || c == 'x' {
			b = append(b, c)
		}
	}
	if len(b) != 10 && len(b) != 13 {
		return ""
	}
	return string(b)
}
