package metadata

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"path"
	"strconv"
	"strings"
)

// extractEPUB reads the OPF package document out of an EPUB archive and
// maps Dublin Core metadata onto Metadata. Covers are extracted from
// whatever manifest item the OPF points at.
func extractEPUB(filePath string) (*Metadata, error) {
	zr, err := zip.OpenReader(filePath)
	if err != nil {
		return nil, fmt.Errorf("open epub: %w", err)
	}
	defer zr.Close()

	opfPath, err := findOPFPath(&zr.Reader)
	if err != nil {
		return nil, err
	}

	opfFile, err := openZipEntry(&zr.Reader, opfPath)
	if err != nil {
		return nil, fmt.Errorf("open opf: %w", err)
	}
	opfBytes, err := io.ReadAll(opfFile)
	opfFile.Close()
	if err != nil {
		return nil, fmt.Errorf("read opf: %w", err)
	}

	var pkg opfPackage
	if err := xml.Unmarshal(opfBytes, &pkg); err != nil {
		return nil, fmt.Errorf("parse opf: %w", err)
	}

	md := &Metadata{
		Title:       firstNonEmpty(pkg.Metadata.Titles),
		Description: firstNonEmpty(pkg.Metadata.Descriptions),
		Language:    firstNonEmpty(pkg.Metadata.Languages),
		Publisher:   firstNonEmpty(pkg.Metadata.Publishers),
		PublishedAt: firstNonEmpty(pkg.Metadata.Dates),
	}
	for _, c := range pkg.Metadata.Creators {
		if c.Text != "" {
			md.Authors = append(md.Authors, c.Text)
		}
	}
	for _, id := range pkg.Metadata.Identifiers {
		if strings.Contains(strings.ToLower(id.Scheme), "isbn") ||
			strings.HasPrefix(strings.ToLower(id.Text), "urn:isbn:") {
			md.ISBN = strings.TrimPrefix(strings.ToLower(id.Text), "urn:isbn:")
			break
		}
	}
	// calibre:series metadata (common convention).
	for _, m := range pkg.Metadata.Meta {
		switch m.Name {
		case "calibre:series":
			md.Series = m.Content
		case "calibre:series_index":
			if f, err := strconv.ParseFloat(m.Content, 64); err == nil {
				md.SeriesIndex = &f
			}
		}
	}

	// Cover extraction: find manifest item marked as cover image,
	// then read its bytes from the archive.
	if coverItem := findCoverItem(&pkg); coverItem != nil {
		coverHref := path.Join(path.Dir(opfPath), coverItem.Href)
		if f, err := openZipEntry(&zr.Reader, coverHref); err == nil {
			data, err := io.ReadAll(f)
			f.Close()
			if err == nil && len(data) > 0 {
				mime := coverItem.MediaType
				if mime == "" {
					mime = http.DetectContentType(data)
				}
				md.Cover = &Cover{Data: data, MIMEType: mime}
			}
		}
	}
	return md, nil
}

// findOPFPath reads META-INF/container.xml and returns the OPF path
// inside the zip.
func findOPFPath(zr *zip.Reader) (string, error) {
	f, err := openZipEntry(zr, "META-INF/container.xml")
	if err != nil {
		return "", fmt.Errorf("open container.xml: %w", err)
	}
	defer f.Close()
	var c containerXML
	if err := xml.NewDecoder(f).Decode(&c); err != nil {
		return "", fmt.Errorf("parse container.xml: %w", err)
	}
	if len(c.Rootfiles) == 0 {
		return "", fmt.Errorf("no rootfile in container.xml")
	}
	return c.Rootfiles[0].FullPath, nil
}

// openZipEntry looks up a path inside the zip, normalising slashes.
func openZipEntry(zr *zip.Reader, name string) (io.ReadCloser, error) {
	name = strings.TrimPrefix(name, "/")
	for _, f := range zr.File {
		if f.Name == name {
			return f.Open()
		}
	}
	return nil, fmt.Errorf("not found in zip: %s", name)
}

// findCoverItem locates the manifest item that represents the cover,
// via either the EPUB3 "cover-image" property or the EPUB2
// <meta name="cover" content="..."/> pointer to a manifest id.
func findCoverItem(pkg *opfPackage) *opfItem {
	// EPUB3
	for i := range pkg.Manifest.Items {
		if strings.Contains(pkg.Manifest.Items[i].Properties, "cover-image") {
			return &pkg.Manifest.Items[i]
		}
	}
	// EPUB2
	var coverID string
	for _, m := range pkg.Metadata.Meta {
		if m.Name == "cover" {
			coverID = m.Content
			break
		}
	}
	if coverID != "" {
		for i := range pkg.Manifest.Items {
			if pkg.Manifest.Items[i].ID == coverID {
				return &pkg.Manifest.Items[i]
			}
		}
	}
	return nil
}

func firstNonEmpty(xs []string) string {
	for _, s := range xs {
		if s := strings.TrimSpace(s); s != "" {
			return s
		}
	}
	return ""
}

// --- XML schemas for container.xml and OPF ---

type containerXML struct {
	XMLName   xml.Name `xml:"container"`
	Rootfiles []struct {
		FullPath  string `xml:"full-path,attr"`
		MediaType string `xml:"media-type,attr"`
	} `xml:"rootfiles>rootfile"`
}

type opfPackage struct {
	XMLName  xml.Name `xml:"package"`
	Metadata struct {
		Titles       []string       `xml:"title"`
		Creators     []opfCreator   `xml:"creator"`
		Descriptions []string       `xml:"description"`
		Languages    []string       `xml:"language"`
		Publishers   []string       `xml:"publisher"`
		Dates        []string       `xml:"date"`
		Identifiers  []opfIdentifer `xml:"identifier"`
		Meta         []opfMeta      `xml:"meta"`
	} `xml:"metadata"`
	Manifest struct {
		Items []opfItem `xml:"item"`
	} `xml:"manifest"`
}

type opfCreator struct {
	Text string `xml:",chardata"`
	Role string `xml:"role,attr"`
}

type opfIdentifer struct {
	Text   string `xml:",chardata"`
	Scheme string `xml:"scheme,attr"`
}

type opfMeta struct {
	Name    string `xml:"name,attr"`
	Content string `xml:"content,attr"`
}

type opfItem struct {
	ID         string `xml:"id,attr"`
	Href       string `xml:"href,attr"`
	MediaType  string `xml:"media-type,attr"`
	Properties string `xml:"properties,attr"`
}
