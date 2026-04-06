package metadata

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
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
	// Try to extract the largest image from the first page as a cover.
	if cover := extractPDFCover(f); cover != nil {
		md.Cover = cover
	}

	// Keywords are often stored as a comma- or semicolon-separated string.
	for _, kw := range info.Keywords {
		for _, sep := range []string{";", ","} {
			if strings.Contains(kw, sep) {
				for _, part := range strings.Split(kw, sep) {
					if s := strings.TrimSpace(part); s != "" {
						md.Subjects = append(md.Subjects, s)
					}
				}
				kw = "" // consumed
				break
			}
		}
		if s := strings.TrimSpace(kw); s != "" {
			md.Subjects = append(md.Subjects, s)
		}
	}
	return md, nil
}

// extractPDFCover extracts the largest image from page 1 as a cover.
func extractPDFCover(rs io.ReadSeeker) *Cover {
	if _, err := rs.Seek(0, io.SeekStart); err != nil {
		return nil
	}
	pages, err := api.ExtractImagesRaw(rs, []string{"1"}, model.NewDefaultConfiguration())
	if err != nil || len(pages) == 0 {
		return nil
	}
	// Find the largest image by pixel area on page 1.
	var best model.Image
	bestArea := 0
	found := false
	for _, pageImages := range pages {
		for _, img := range pageImages {
			area := img.Width * img.Height
			if area > bestArea {
				best = img
				bestArea = area
				found = true
			}
		}
	}
	if !found || bestArea < 100*100 {
		return nil // skip tiny images (icons, decorations)
	}
	data, err := io.ReadAll(&best)
	if err != nil || len(data) == 0 {
		return nil
	}
	mime := "image/" + strings.ToLower(best.FileType)
	if best.FileType == "" {
		mime = "image/jpeg"
	}
	return &Cover{Data: data, MIMEType: mime}
}
