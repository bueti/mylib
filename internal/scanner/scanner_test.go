package scanner

import (
	"archive/zip"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/bueti/mylib/internal/covers"
	"github.com/bueti/mylib/internal/db"
	"github.com/bueti/mylib/internal/library"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScanner_AddUpdateRemove(t *testing.T) {
	ctx := context.Background()
	dataDir := t.TempDir()
	libRoot := t.TempDir()

	conn, err := db.Open(dataDir)
	require.NoError(t, err)
	defer conn.Close()
	store := library.New(conn)

	coverCache, err := covers.New(dataDir)
	require.NoError(t, err)

	sc := New(store, []string{libRoot}, coverCache)

	// 1. Empty root: scan produces zero books.
	jobID, err := sc.ScanAll(ctx)
	require.NoError(t, err)
	waitForScan(t, store, jobID)

	books, total, err := store.ListBooks(ctx, library.BookFilter{})
	require.NoError(t, err)
	assert.Equal(t, 0, total)
	assert.Empty(t, books)

	// 2. Add one epub → scan → one book.
	writeEPUB(t, filepath.Join(libRoot, "one.epub"), "Title One", "Author One")

	jobID, err = sc.ScanAll(ctx)
	require.NoError(t, err)
	waitForScan(t, store, jobID)

	books, total, err = store.ListBooks(ctx, library.BookFilter{})
	require.NoError(t, err)
	require.Equal(t, 1, total)
	require.Len(t, books, 1)
	assert.Equal(t, "Title One", books[0].Title)
	require.Len(t, books[0].Authors, 1)
	assert.Equal(t, "Author One", books[0].Authors[0].Name)

	job, err := store.GetScanJob(ctx, jobID)
	require.NoError(t, err)
	assert.Equal(t, "done", job.Status)
	assert.Equal(t, 1, job.FilesAdded)
	assert.Equal(t, 0, job.FilesRemoved)

	// 3. Modify the file → scan → still one book, counted as updated.
	time.Sleep(1100 * time.Millisecond) // ensure mtime granularity changes
	writeEPUB(t, filepath.Join(libRoot, "one.epub"), "Title One Revised", "Author One")

	jobID, err = sc.ScanAll(ctx)
	require.NoError(t, err)
	waitForScan(t, store, jobID)

	books, total, err = store.ListBooks(ctx, library.BookFilter{})
	require.NoError(t, err)
	require.Equal(t, 1, total)
	assert.Equal(t, "Title One Revised", books[0].Title)

	job, err = store.GetScanJob(ctx, jobID)
	require.NoError(t, err)
	assert.Equal(t, 1, job.FilesUpdated)

	// 4. Delete the file → scan → zero active books, one counted removed.
	require.NoError(t, os.Remove(filepath.Join(libRoot, "one.epub")))

	jobID, err = sc.ScanAll(ctx)
	require.NoError(t, err)
	waitForScan(t, store, jobID)

	_, total, err = store.ListBooks(ctx, library.BookFilter{})
	require.NoError(t, err)
	assert.Equal(t, 0, total)

	job, err = store.GetScanJob(ctx, jobID)
	require.NoError(t, err)
	assert.Equal(t, 1, job.FilesRemoved)
}

func TestScanner_FTSSearch(t *testing.T) {
	ctx := context.Background()
	dataDir := t.TempDir()
	libRoot := t.TempDir()

	conn, err := db.Open(dataDir)
	require.NoError(t, err)
	defer conn.Close()
	store := library.New(conn)
	coverCache, err := covers.New(dataDir)
	require.NoError(t, err)
	sc := New(store, []string{libRoot}, coverCache)

	writeEPUB(t, filepath.Join(libRoot, "a.epub"), "A Wizard of Earthsea", "Ursula Le Guin")
	writeEPUB(t, filepath.Join(libRoot, "b.epub"), "Dune", "Frank Herbert")
	writeEPUB(t, filepath.Join(libRoot, "c.epub"), "The Dispossessed", "Ursula Le Guin")

	jobID, err := sc.ScanAll(ctx)
	require.NoError(t, err)
	waitForScan(t, store, jobID)

	_, total, err := store.ListBooks(ctx, library.BookFilter{})
	require.NoError(t, err)
	require.Equal(t, 3, total)

	// Full-text match on author.
	books, total, err := store.ListBooks(ctx, library.BookFilter{Query: "guin"})
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	require.Len(t, books, 2)

	// Prefix match on title.
	books, total, err = store.ListBooks(ctx, library.BookFilter{Query: "wiz"})
	require.NoError(t, err)
	require.Equal(t, 1, total)
	require.Len(t, books, 1)
	assert.Equal(t, "A Wizard of Earthsea", books[0].Title)
	assert.Equal(t, "Wizard of Earthsea", books[0].SortTitle)

	// SortTitle strips the leading article on another book too.
	books, total, err = store.ListBooks(ctx, library.BookFilter{Query: "dispossessed"})
	require.NoError(t, err)
	require.Equal(t, 1, total)
	require.Len(t, books, 1)
	assert.Equal(t, "Dispossessed", books[0].SortTitle)
}

// --- helpers ---

// waitForScan polls the scan job until finished (bounded).
func waitForScan(t *testing.T, store *library.Store, jobID int64) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		job, err := store.GetScanJob(context.Background(), jobID)
		require.NoError(t, err)
		if job.FinishedAt != nil {
			require.Equal(t, "done", job.Status, "scan failed: %s", job.Error)
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("scan %d did not finish", jobID)
}

// writeEPUB writes a minimal valid EPUB with one title + author to path.
func writeEPUB(t *testing.T, path, title, author string) {
	t.Helper()
	f, err := os.Create(path)
	require.NoError(t, err)
	defer f.Close()
	zw := zip.NewWriter(f)

	mt, err := zw.Create("mimetype")
	require.NoError(t, err)
	_, err = mt.Write([]byte("application/epub+zip"))
	require.NoError(t, err)

	c, err := zw.Create("META-INF/container.xml")
	require.NoError(t, err)
	_, err = c.Write([]byte(`<?xml version="1.0"?><container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container"><rootfiles><rootfile full-path="content.opf" media-type="application/oebps-package+xml"/></rootfiles></container>`))
	require.NoError(t, err)

	opf, err := zw.Create("content.opf")
	require.NoError(t, err)
	_, err = opf.Write([]byte(`<?xml version="1.0"?><package xmlns="http://www.idpf.org/2007/opf" version="3.0"><metadata xmlns:dc="http://purl.org/dc/elements/1.1/"><dc:title>` + title + `</dc:title><dc:creator>` + author + `</dc:creator></metadata><manifest></manifest></package>`))
	require.NoError(t, err)

	require.NoError(t, zw.Close())
}
