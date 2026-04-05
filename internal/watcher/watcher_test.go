package watcher

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/bueti/mylib/internal/covers"
	"github.com/bueti/mylib/internal/db"
	"github.com/bueti/mylib/internal/library"
	"github.com/bueti/mylib/internal/scanner"
	"github.com/stretchr/testify/require"
)

func TestWatcher_DetectsNewFiles(t *testing.T) {
	dataDir := t.TempDir()
	libRoot := t.TempDir()

	conn, err := db.Open(dataDir)
	require.NoError(t, err)
	defer conn.Close()
	store := library.New(conn)
	coverCache, err := covers.New(dataDir)
	require.NoError(t, err)
	sc := scanner.New(store, []string{libRoot}, coverCache)

	w, err := New(sc, []string{libRoot}, 200*time.Millisecond)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = w.Run(ctx) }()

	// Give fsnotify a tick to settle.
	time.Sleep(100 * time.Millisecond)

	// Drop a .txt file — not an ebook, but it'll trigger the watcher.
	// The scanner will ignore it (DetectFormat returns "") but a
	// scan_job will still be created.
	require.NoError(t, os.WriteFile(filepath.Join(libRoot, "poke.txt"), []byte("x"), 0o644))

	// Wait for the debounce + scan to run.
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		rows, err := conn.QueryContext(ctx, `SELECT COUNT(*) FROM scan_jobs`)
		require.NoError(t, err)
		var n int
		rows.Next()
		require.NoError(t, rows.Scan(&n))
		rows.Close()
		if n >= 1 {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("watcher did not trigger a scan within 3s")
}
