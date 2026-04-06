// Package metadata extracts book metadata (title, authors, series, cover…)
// from EPUB and PDF files. It dispatches to format-specific extractors
// based on file extension and falls back to filename heuristics when
// embedded metadata is absent.
package metadata

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

// Format identifies a supported ebook format.
type Format string

const (
	FormatEPUB Format = "epub"
	FormatPDF  Format = "pdf"
	FormatMOBI Format = "mobi"
	FormatAZW3 Format = "azw3"
)

// Metadata is the extracted bibliographic data for a single book.
// All fields are optional except Title (which always has a value,
// even if it had to be derived from the filename).
type Metadata struct {
	Title       string
	Subtitle    string
	Authors     []string
	Series      string
	SeriesIndex *float64
	Description string
	Language    string
	ISBN        string
	Publisher   string
	PublishedAt string   // ISO date or year
	Subjects    []string // genre/topic tags from dc:subject or keywords
	// Cover is the raw image bytes and MIME type, or nil if no cover
	// was found.
	Cover *Cover
}

// Cover holds a raw cover image.
type Cover struct {
	Data     []byte
	MIMEType string
}

// DetectFormat returns the Format for a given file path, or "" if the
// extension is not recognised.
func DetectFormat(path string) Format {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".epub":
		return FormatEPUB
	case ".pdf":
		return FormatPDF
	case ".mobi":
		return FormatMOBI
	case ".azw3", ".azw":
		return FormatAZW3
	default:
		return ""
	}
}

// Extract reads metadata from the file at path. The returned Metadata
// always has a non-empty Title; if no embedded title is found, one is
// derived from the filename.
func Extract(path string) (*Metadata, error) {
	format := DetectFormat(path)
	if format == "" {
		return nil, fmt.Errorf("unsupported format: %s", path)
	}

	var (
		md  *Metadata
		err error
	)
	switch format {
	case FormatEPUB:
		md, err = extractEPUB(path)
	case FormatPDF:
		md, err = extractPDF(path)
	default:
		// MOBI/AZW3: filename-only for v0.1.
		md = &Metadata{}
	}
	if err != nil {
		// Fall through to filename heuristics on parser errors rather
		// than dropping the book entirely.
		md = &Metadata{}
	}
	if md == nil {
		md = &Metadata{}
	}
	applyFilenameFallback(md, path)
	return md, nil
}

// applyFilenameFallback fills in Title and Authors from filename patterns
// like "Author - Title.epub" when those fields are empty.
var filenamePattern = regexp.MustCompile(`^\s*(.+?)\s*-\s*(.+?)\s*$`)

func applyFilenameFallback(md *Metadata, path string) {
	base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	if md.Title == "" || len(md.Authors) == 0 {
		if m := filenamePattern.FindStringSubmatch(base); m != nil {
			if len(md.Authors) == 0 {
				md.Authors = []string{strings.TrimSpace(m[1])}
			}
			if md.Title == "" {
				md.Title = strings.TrimSpace(m[2])
			}
		}
	}
	if md.Title == "" {
		md.Title = base
	}
}
