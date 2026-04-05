// Package config loads mylib runtime settings from the environment.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Config holds all runtime settings. All fields are populated from env vars
// prefixed with MYLIB_.
type Config struct {
	// LibraryRoots is the list of directories to scan for ebooks/PDFs.
	LibraryRoots []string
	// DataDir is where the SQLite DB and extracted covers are stored.
	DataDir string
	// Listen is the HTTP listen address, e.g. ":8080".
	Listen string
	// ScanInterval is how often to re-scan roots. Zero disables periodic scans.
	ScanInterval time.Duration
	// LogLevel is one of debug, info, warn, error.
	LogLevel string
}

// Load reads configuration from the environment and validates it.
func Load() (*Config, error) {
	rootsRaw := os.Getenv("MYLIB_LIBRARY_ROOTS")
	if rootsRaw == "" {
		return nil, fmt.Errorf("MYLIB_LIBRARY_ROOTS is required")
	}
	var roots []string
	for _, p := range strings.Split(rootsRaw, string(os.PathListSeparator)) {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		abs, err := filepath.Abs(p)
		if err != nil {
			return nil, fmt.Errorf("resolve root %q: %w", p, err)
		}
		roots = append(roots, abs)
	}
	if len(roots) == 0 {
		return nil, fmt.Errorf("MYLIB_LIBRARY_ROOTS resolved to no paths")
	}

	dataDir := envOr("MYLIB_DATA_DIR", "./data")
	dataDir, err := filepath.Abs(dataDir)
	if err != nil {
		return nil, fmt.Errorf("resolve data dir: %w", err)
	}

	interval, err := time.ParseDuration(envOr("MYLIB_SCAN_INTERVAL", "10m"))
	if err != nil {
		return nil, fmt.Errorf("parse MYLIB_SCAN_INTERVAL: %w", err)
	}

	return &Config{
		LibraryRoots: roots,
		DataDir:      dataDir,
		Listen:       envOr("MYLIB_LISTEN", ":8080"),
		ScanInterval: interval,
		LogLevel:     envOr("MYLIB_LOG_LEVEL", "info"),
	}, nil
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
