// Package library holds the core domain types and the SQLite-backed
// store that the rest of mylib reads from and writes to.
package library

import "time"

// Book is the canonical view of one book in the library.
type Book struct {
	ID          int64
	ContentHash string
	Path        string
	Format      string
	SizeBytes   int64
	MTime       time.Time
	Title       string
	SortTitle   string
	Subtitle    string
	Description string
	SeriesID    *int64
	SeriesName  string
	SeriesIndex *float64
	Language    string
	ISBN        string
	Publisher   string
	PublishedAt string
	AddedAt     time.Time
	CoverPath   string
	Authors     []Author
	Tags        []string
	DeletedAt   *time.Time
}

// Author is a single author row.
type Author struct {
	ID       int64
	Name     string
	SortName string
}

// Series is a single series row.
type Series struct {
	ID   int64
	Name string
}

// BookFilter narrows a list query against the store.
type BookFilter struct {
	Query        string // full-text search over title/subtitle/authors/series/description/tags
	AuthorID     *int64
	SeriesID     *int64
	CollectionID *int64
	Tag          string
	Format       string
	Sort         string // "title", "added", "-added" (prefix '-' reverses)
	Limit        int
	Offset       int
}

// ScanJob is the state of a single scan run.
type ScanJob struct {
	ID           int64
	Root         string
	StartedAt    time.Time
	FinishedAt   *time.Time
	Status       string // "running", "done", "error"
	FilesSeen    int
	FilesAdded   int
	FilesUpdated int
	FilesRemoved int
	Error        string
}
