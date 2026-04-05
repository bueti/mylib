package library

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindDuplicates_ByISBN(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)

	// Two copies with the same ISBN, and an unrelated book.
	_, err := s.UpsertBook(ctx, &Book{
		ContentHash: "h1", Path: "/a.epub", Format: "epub", SizeBytes: 1,
		Title: "Dune", SortTitle: "Dune", ISBN: "978-0441013593",
		Authors: []Author{{Name: "Frank Herbert", SortName: "Herbert, Frank"}},
	})
	require.NoError(t, err)
	_, err = s.UpsertBook(ctx, &Book{
		ContentHash: "h2", Path: "/b.epub", Format: "epub", SizeBytes: 1,
		Title: "Dune (special edition)", SortTitle: "Dune special edition", ISBN: "9780441013593",
		Authors: []Author{{Name: "Frank Herbert", SortName: "Herbert, Frank"}},
	})
	require.NoError(t, err)
	_, err = s.UpsertBook(ctx, &Book{
		ContentHash: "h3", Path: "/c.epub", Format: "epub", SizeBytes: 1,
		Title: "Foundation", SortTitle: "Foundation", ISBN: "9780553293357",
		Authors: []Author{{Name: "Isaac Asimov", SortName: "Asimov, Isaac"}},
	})
	require.NoError(t, err)

	groups, err := s.FindDuplicates(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, groups)
	// Expect one ISBN group containing 2 books.
	var isbnGroup *DuplicateGroup
	for _, g := range groups {
		if g.Reason == "isbn" {
			isbnGroup = g
			break
		}
	}
	require.NotNil(t, isbnGroup)
	assert.Len(t, isbnGroup.Books, 2)
}

func TestFindDuplicates_ByTitle(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)

	// Same title + author, different ISBN — caught by title strategy.
	_, err := s.UpsertBook(ctx, &Book{
		ContentHash: "h1", Path: "/a.epub", Format: "epub", SizeBytes: 1,
		Title: "The Left Hand of Darkness", SortTitle: "Left Hand of Darkness",
		Authors: []Author{{Name: "Ursula K. Le Guin", SortName: "Le Guin, Ursula K."}},
	})
	require.NoError(t, err)
	_, err = s.UpsertBook(ctx, &Book{
		ContentHash: "h2", Path: "/b.epub", Format: "epub", SizeBytes: 1,
		Title: "The Left Hand of Darkness!", SortTitle: "Left Hand of Darkness",
		Authors: []Author{{Name: "Ursula K. Le Guin", SortName: "Le Guin, Ursula K."}},
	})
	require.NoError(t, err)

	groups, err := s.FindDuplicates(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, groups)
	var titleGroup *DuplicateGroup
	for _, g := range groups {
		if g.Reason == "title" {
			titleGroup = g
			break
		}
	}
	require.NotNil(t, titleGroup)
	assert.Len(t, titleGroup.Books, 2)
}

func TestFindDuplicates_None(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)
	_, err := s.UpsertBook(ctx, &Book{
		ContentHash: "h1", Path: "/a.epub", Format: "epub", SizeBytes: 1,
		Title: "Alpha", SortTitle: "Alpha",
		Authors: []Author{{Name: "A", SortName: "A"}},
	})
	require.NoError(t, err)
	_, err = s.UpsertBook(ctx, &Book{
		ContentHash: "h2", Path: "/b.epub", Format: "epub", SizeBytes: 1,
		Title: "Beta", SortTitle: "Beta",
		Authors: []Author{{Name: "B", SortName: "B"}},
	})
	require.NoError(t, err)
	groups, err := s.FindDuplicates(ctx)
	require.NoError(t, err)
	assert.Empty(t, groups)
}
