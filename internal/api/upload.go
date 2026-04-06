package api

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bueti/mylib/internal/enrich"
	"github.com/bueti/mylib/internal/library"
	"github.com/bueti/mylib/internal/metadata"
	"github.com/go-chi/chi/v5"
)

const maxUploadSize = 100 << 20 // 100 MB per request

// registerUpload wires the book upload endpoint.
func registerUpload(r chi.Router, d Deps) {
	uploadDir := filepath.Join(d.LibraryRoot, "uploads")

	r.With(RequireAuth(d.Store)).Post("/api/books/upload", func(w http.ResponseWriter, req *http.Request) {
		req.Body = http.MaxBytesReader(w, req.Body, maxUploadSize)
		if err := req.ParseMultipartForm(maxUploadSize); err != nil {
			http.Error(w, "request too large (max 100MB)", http.StatusRequestEntityTooLarge)
			return
		}
		defer req.MultipartForm.RemoveAll()

		files := req.MultipartForm.File["files"]
		if len(files) == 0 {
			http.Error(w, "no files in 'files' field", http.StatusBadRequest)
			return
		}

		// Ensure upload directory exists.
		if err := os.MkdirAll(uploadDir, 0o755); err != nil {
			http.Error(w, "could not create upload directory", http.StatusInternalServerError)
			return
		}

		type result struct {
			Book  *BookDTO `json:"book,omitempty"`
			File  string   `json:"file"`
			Error string   `json:"error,omitempty"`
		}
		var results []result

		for _, fh := range files {
			res := result{File: fh.Filename}

			// Validate format.
			format := metadata.DetectFormat(fh.Filename)
			if format == "" {
				res.Error = "unsupported format (accepted: epub, pdf, mobi, azw3)"
				results = append(results, res)
				continue
			}

			// Write to disk.
			diskPath, err := writeUploadedFile(uploadDir, fh)
			if err != nil {
				res.Error = "write failed: " + err.Error()
				results = append(results, res)
				continue
			}

			// Process: hash → metadata → upsert → cover → enrich queue.
			book, err := processUploadedFile(req, d, diskPath, string(format))
			if err != nil {
				res.Error = "process failed: " + err.Error()
				results = append(results, res)
				continue
			}
			dto := toBookDTO(book)
			res.Book = &dto
			results = append(results, res)
		}

		writeJSON(w, http.StatusOK, map[string]any{"results": results})
	})
}

// writeUploadedFile saves a multipart file to the upload directory,
// avoiding name collisions.
func writeUploadedFile(dir string, fh *multipart.FileHeader) (string, error) {
	src, err := fh.Open()
	if err != nil {
		return "", err
	}
	defer src.Close()

	name := sanitizeFilename(fh.Filename)
	destPath := filepath.Join(dir, name)

	// Avoid collisions.
	if _, err := os.Stat(destPath); err == nil {
		ext := filepath.Ext(name)
		base := strings.TrimSuffix(name, ext)
		for i := 2; i < 1000; i++ {
			candidate := fmt.Sprintf("%s (%d)%s", base, i, ext)
			destPath = filepath.Join(dir, candidate)
			if _, err := os.Stat(destPath); os.IsNotExist(err) {
				break
			}
		}
	}

	dst, err := os.Create(destPath)
	if err != nil {
		return "", err
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		os.Remove(destPath)
		return "", err
	}
	return destPath, nil
}

// processUploadedFile runs the same pipeline as the scanner on a
// single file: hash, extract metadata, upsert, cache cover, queue
// enrichment.
func processUploadedFile(req *http.Request, d Deps, path, format string) (*library.Book, error) {
	st, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	hash, err := hashFile(path, st.Size())
	if err != nil {
		return nil, fmt.Errorf("hash: %w", err)
	}

	md, err := metadata.Extract(path)
	if err != nil {
		return nil, fmt.Errorf("metadata: %w", err)
	}

	book := &library.Book{
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
		book.Authors = append(book.Authors, library.Author{
			Name:     name,
			SortName: library.SortName(name),
		})
	}
	book.Tags = append(book.Tags, enrich.NormalizeSubjects(md.Subjects)...)

	// Cache cover.
	if md.Cover != nil && d.Covers != nil {
		if rel, err := d.Covers.Store(hash, md.Cover.Data, md.Cover.MIMEType); err == nil {
			book.CoverPath = rel
		}
	}

	id, err := d.Store.UpsertBook(req.Context(), book)
	if err != nil {
		return nil, fmt.Errorf("upsert: %w", err)
	}

	// Queue async enrichment.
	if d.EnrichQueue != nil && (book.ISBN != "" || book.Title != "") {
		select {
		case d.EnrichQueue <- id:
		default:
		}
	}

	slog.Info("uploaded book", "id", id, "title", book.Title, "path", path)

	return d.Store.GetBook(req.Context(), id)
}

// hashFile produces a content fingerprint. Duplicated from scanner
// (which is unexported) — same algorithm: full hash for small files,
// first+last 64KB + length for large ones.
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

// sanitizeFilename strips dangerous characters from an upload filename.
func sanitizeFilename(name string) string {
	// Use only the base name (strip any directory components).
	name = filepath.Base(name)
	// Remove null bytes, path separators, leading dots.
	var b strings.Builder
	for _, r := range name {
		switch {
		case r == 0:
			continue
		case r == '/' || r == '\\':
			b.WriteRune('_')
		default:
			b.WriteRune(r)
		}
	}
	name = strings.TrimLeft(b.String(), ".")
	name = strings.TrimSpace(name)
	if name == "" {
		name = "upload-" + fmt.Sprintf("%d", time.Now().UnixMilli())
	}
	return name
}
