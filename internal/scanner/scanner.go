// Package scanner walks configured library roots, upserting book
// records from extracted metadata and soft-deleting books whose files
// have disappeared.
package scanner

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/bueti/mylib/internal/covers"
	"github.com/bueti/mylib/internal/library"
	"github.com/bueti/mylib/internal/metadata"
)

// Scanner walks library roots and keeps the store in sync with the
// filesystem. Only one scan runs at a time; concurrent Run calls share
// the in-flight job.
type Scanner struct {
	store  *library.Store
	roots  []string
	covers *covers.Cache

	mu       sync.Mutex
	inFlight int64 // job id; 0 when idle
}

// New builds a Scanner. dataDir is where covers are written.
func New(store *library.Store, roots []string, coverCache *covers.Cache) *Scanner {
	return &Scanner{store: store, roots: roots, covers: coverCache}
}

// ScanAll runs a full scan across all configured roots and returns
// the scan job id.
func (s *Scanner) ScanAll(ctx context.Context) (int64, error) {
	// Single root semantics: we create one scan_job per invocation,
	// tracking multi-root scans as a combined root string.
	s.mu.Lock()
	if s.inFlight != 0 {
		id := s.inFlight
		s.mu.Unlock()
		return id, nil
	}
	rootLabel := strings.Join(s.roots, ":")
	id, err := s.store.CreateScanJob(ctx, rootLabel)
	if err != nil {
		s.mu.Unlock()
		return 0, err
	}
	s.inFlight = id
	s.mu.Unlock()

	go s.runScan(context.WithoutCancel(ctx), id)
	return id, nil
}

// runScan is the actual scanning work, run in a goroutine.
func (s *Scanner) runScan(ctx context.Context, jobID int64) {
	defer func() {
		s.mu.Lock()
		s.inFlight = 0
		s.mu.Unlock()
	}()

	job := library.ScanJob{Status: "done"}
	start := time.Now()
	for _, root := range s.roots {
		if err := s.scanRoot(ctx, root, &job); err != nil {
			slog.Error("scan root failed", "root", root, "err", err)
			job.Status = "error"
			job.Error = err.Error()
			break
		}
	}
	if err := s.store.FinishScanJob(ctx, jobID, job); err != nil {
		slog.Error("finish scan job", "err", err)
	}
	slog.Info("scan complete",
		"job_id", jobID, "status", job.Status, "duration", time.Since(start),
		"seen", job.FilesSeen, "added", job.FilesAdded,
		"updated", job.FilesUpdated, "removed", job.FilesRemoved,
	)
}

// scanRoot walks one root directory.
func (s *Scanner) scanRoot(ctx context.Context, root string, job *library.ScanJob) error {
	info, err := os.Stat(root)
	if err != nil {
		return fmt.Errorf("stat root: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("root is not a directory: %s", root)
	}

	existing, err := s.store.ListActivePaths(ctx, root)
	if err != nil {
		return fmt.Errorf("list active paths: %w", err)
	}
	seen := make(map[string]struct{}, len(existing))

	err = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			slog.Warn("walk error", "path", path, "err", err)
			return nil // skip unreadable entries; keep walking
		}
		if d.IsDir() {
			return nil
		}
		if metadata.DetectFormat(path) == "" {
			return nil
		}
		job.FilesSeen++
		seen[path] = struct{}{}
		return s.processFile(ctx, path, existing, job)
	})
	if err != nil {
		return err
	}

	// Soft-delete books whose files weren't seen on this walk.
	for path, id := range existing {
		if _, ok := seen[path]; ok {
			continue
		}
		if err := s.store.SoftDelete(ctx, id); err != nil {
			slog.Warn("soft delete failed", "path", path, "err", err)
			continue
		}
		job.FilesRemoved++
	}
	return nil
}

// processFile handles one file: stat → optional skip → hash → extract
// metadata → upsert.
func (s *Scanner) processFile(ctx context.Context, path string, existing map[string]int64, job *library.ScanJob) error {
	st, err := os.Stat(path)
	if err != nil {
		slog.Warn("stat failed", "path", path, "err", err)
		return nil
	}

	// Fast path: same path, same size + mtime → nothing changed.
	if _, ok := existing[path]; ok {
		prior, err := s.store.GetBookByPath(ctx, path)
		if err == nil && prior.DeletedAt == nil &&
			prior.SizeBytes == st.Size() && prior.MTime.Unix() == st.ModTime().Unix() {
			return nil
		}
	}

	hash, err := hashFile(path, st.Size())
	if err != nil {
		slog.Warn("hash failed", "path", path, "err", err)
		return nil
	}

	md, err := metadata.Extract(path)
	if err != nil {
		slog.Warn("metadata extract failed", "path", path, "err", err)
		return nil
	}

	book := buildBook(path, st, hash, md, string(metadata.DetectFormat(path)))
	if md.Cover != nil && s.covers != nil {
		coverPath, err := s.covers.Store(hash, md.Cover.Data, md.Cover.MIMEType)
		if err != nil {
			slog.Warn("cover store failed", "path", path, "err", err)
		} else {
			book.CoverPath = coverPath
		}
	}

	// Detect add vs update by checking if this path was previously known.
	_, prevErr := s.store.GetBookByPath(ctx, path)
	isUpdate := prevErr == nil
	if _, err := s.store.UpsertBook(ctx, book); err != nil {
		return fmt.Errorf("upsert %s: %w", path, err)
	}
	if isUpdate {
		job.FilesUpdated++
	} else {
		job.FilesAdded++
	}
	return nil
}

// buildBook constructs a library.Book from file info and extracted metadata.
func buildBook(path string, st os.FileInfo, hash string, md *metadata.Metadata, format string) *library.Book {
	b := &library.Book{
		ContentHash: hash,
		Path:        path,
		Format:      format,
		SizeBytes:   st.Size(),
		MTime:       st.ModTime(),
		Title:       md.Title,
		SortTitle:   library.SortTitle(md.Title),
		Subtitle:    md.Subtitle,
		Description: md.Description,
		SeriesName:  md.Series,
		SeriesIndex: md.SeriesIndex,
		Language:    md.Language,
		ISBN:        md.ISBN,
		Publisher:   md.Publisher,
		PublishedAt: md.PublishedAt,
	}
	for _, name := range md.Authors {
		b.Authors = append(b.Authors, library.Author{
			Name:     name,
			SortName: library.SortName(name),
		})
	}
	return b
}

// hashFile returns a content fingerprint for the file. For small files
// (<256KB) we hash the whole thing; for larger ones we hash the first
// and last 64KB plus the length — fast and collision-resistant enough
// for library dedup.
func hashFile(path string, size int64) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	const chunk = 64 * 1024
	h := sha256.New()
	if size <= 4*chunk {
		if _, err := io.Copy(h, f); err != nil {
			return "", err
		}
	} else {
		buf := make([]byte, chunk)
		if _, err := io.ReadFull(f, buf); err != nil {
			return "", err
		}
		h.Write(buf)
		if _, err := f.Seek(size-chunk, io.SeekStart); err != nil {
			return "", err
		}
		if _, err := io.ReadFull(f, buf); err != nil {
			return "", err
		}
		h.Write(buf)
		fmt.Fprintf(h, "%d", size)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
