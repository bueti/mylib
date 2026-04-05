// Package watcher uses fsnotify to kick off library scans shortly
// after files in watched roots change, so new books appear in the UI
// without waiting for the periodic ticker.
package watcher

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/bueti/mylib/internal/scanner"
	"github.com/fsnotify/fsnotify"
)

// Watcher observes one or more library roots. Events are coalesced
// into a single scan per debounce window.
type Watcher struct {
	scanner  *scanner.Scanner
	roots    []string
	debounce time.Duration
	fsw      *fsnotify.Watcher
}

// New constructs a Watcher. debounce is how long to wait for more
// events after the first one before kicking off a scan. Recommended
// values: 1–3 seconds.
func New(sc *scanner.Scanner, roots []string, debounce time.Duration) (*Watcher, error) {
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	return &Watcher{scanner: sc, roots: roots, debounce: debounce, fsw: fsw}, nil
}

// Run watches the configured roots recursively until ctx is canceled.
func (w *Watcher) Run(ctx context.Context) error {
	defer w.fsw.Close()

	for _, root := range w.roots {
		if err := w.addRecursive(root); err != nil {
			slog.Warn("watch root failed", "root", root, "err", err)
		}
	}

	var (
		timer  *time.Timer
		timerC <-chan time.Time
	)
	reset := func() {
		if timer != nil {
			timer.Stop()
		}
		timer = time.NewTimer(w.debounce)
		timerC = timer.C
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case ev, ok := <-w.fsw.Events:
			if !ok {
				return nil
			}
			slog.Debug("fs event", "op", ev.Op.String(), "name", ev.Name)
			// Subscribe to new subdirectories as they appear.
			if ev.Op.Has(fsnotify.Create) {
				if info, err := os.Stat(ev.Name); err == nil && info.IsDir() {
					_ = w.addRecursive(ev.Name)
				}
			}
			reset()
		case err, ok := <-w.fsw.Errors:
			if !ok {
				return nil
			}
			slog.Warn("fs watch error", "err", err)
		case <-timerC:
			slog.Info("fs change detected — rescanning")
			if _, err := w.scanner.ScanAll(ctx); err != nil {
				slog.Warn("scan trigger failed", "err", err)
			}
			timerC = nil
		}
	}
}

// Close releases the underlying fsnotify watcher.
func (w *Watcher) Close() error { return w.fsw.Close() }

// addRecursive adds path and all its subdirectories to the watcher.
// Files are not watched directly; we rely on their parent-directory
// events.
func (w *Watcher) addRecursive(root string) error {
	return filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // skip unreadable entries
		}
		if !d.IsDir() {
			return nil
		}
		if err := w.fsw.Add(path); err != nil {
			slog.Warn("add watch failed", "path", path, "err", err)
		}
		return nil
	})
}
