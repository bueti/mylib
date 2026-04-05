// Package covers caches extracted book cover images on disk, keyed by
// book content hash. Files are served directly by the HTTP layer.
package covers

import (
	"fmt"
	"os"
	"path/filepath"
)

// Cache writes cover images under dataDir/covers/ and returns paths
// relative to dataDir.
type Cache struct {
	root string // absolute path to covers/ dir
}

// New creates (if necessary) and returns a Cache rooted under dataDir.
func New(dataDir string) (*Cache, error) {
	root := filepath.Join(dataDir, "covers")
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, fmt.Errorf("create covers dir: %w", err)
	}
	return &Cache{root: root}, nil
}

// Store writes cover bytes for a book and returns the relative path
// ("covers/ab/abcdef….jpg") that should be stored in books.cover_path.
func (c *Cache) Store(contentHash string, data []byte, mimeType string) (string, error) {
	if len(contentHash) < 2 {
		return "", fmt.Errorf("content hash too short")
	}
	shard := contentHash[:2]
	ext := extFromMIME(mimeType)
	rel := filepath.Join("covers", shard, contentHash+ext)
	abs := filepath.Join(filepath.Dir(c.root), rel)
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(abs, data, 0o644); err != nil {
		return "", err
	}
	return rel, nil
}

// AbsPath returns the absolute filesystem path for a cover's relative path.
func (c *Cache) AbsPath(rel string) string {
	return filepath.Join(filepath.Dir(c.root), rel)
}

func extFromMIME(m string) string {
	switch m {
	case "image/jpeg", "image/jpg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	default:
		return ".img"
	}
}
