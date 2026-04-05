package metadata

import (
	"fmt"
	"os"

	"github.com/pdfcpu/pdfcpu/pkg/api"
)

// extractPDF reads the Info dictionary and XMP metadata from a PDF via
// pdfcpu. Cover extraction (first-page render) is deferred to post-MVP.
func extractPDF(filePath string) (*Metadata, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("open pdf: %w", err)
	}
	defer f.Close()
	info, err := api.PDFInfo(f, filePath, nil, false, nil)
	if err != nil {
		return nil, fmt.Errorf("pdf info: %w", err)
	}
	md := &Metadata{
		Title:     info.Title,
		Publisher: info.Producer,
	}
	if info.Author != "" {
		md.Authors = []string{info.Author}
	}
	if info.CreationDate != "" {
		md.PublishedAt = info.CreationDate
	}
	return md, nil
}
