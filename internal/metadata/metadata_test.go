package metadata

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectFormat(t *testing.T) {
	cases := map[string]Format{
		"book.epub":             FormatEPUB,
		"BOOK.EPUB":             FormatEPUB,
		"report.pdf":            FormatPDF,
		"thing.mobi":            FormatMOBI,
		"thing.azw3":            FormatAZW3,
		"thing.azw":             FormatAZW3,
		"something.txt":         Format(""),
		"/path/to/novel.epub":   FormatEPUB,
	}
	for path, want := range cases {
		assert.Equal(t, want, DetectFormat(path), "path=%s", path)
	}
}

func TestExtract_EPUB_FullMetadata(t *testing.T) {
	path := buildEPUB(t, epubSpec{
		OPF: `<?xml version="1.0"?>
<package xmlns="http://www.idpf.org/2007/opf" version="3.0">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
    <dc:title>A Wizard of Earthsea</dc:title>
    <dc:creator>Ursula K. Le Guin</dc:creator>
    <dc:language>en</dc:language>
    <dc:publisher>Parnassus Press</dc:publisher>
    <dc:date>1968</dc:date>
    <dc:identifier scheme="ISBN">9780553262506</dc:identifier>
    <dc:description>A boy with a gift for magic.</dc:description>
    <meta name="calibre:series" content="Earthsea"/>
    <meta name="calibre:series_index" content="1"/>
    <meta name="cover" content="cover-img"/>
  </metadata>
  <manifest>
    <item id="cover-img" href="cover.png" media-type="image/png"/>
  </manifest>
</package>`,
		Cover: []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}, // PNG magic
	})

	md, err := Extract(path)
	require.NoError(t, err)
	assert.Equal(t, "A Wizard of Earthsea", md.Title)
	require.Len(t, md.Authors, 1)
	assert.Equal(t, "Ursula K. Le Guin", md.Authors[0])
	assert.Equal(t, "en", md.Language)
	assert.Equal(t, "Parnassus Press", md.Publisher)
	assert.Equal(t, "1968", md.PublishedAt)
	assert.Equal(t, "9780553262506", md.ISBN)
	assert.Equal(t, "A boy with a gift for magic.", md.Description)
	assert.Equal(t, "Earthsea", md.Series)
	require.NotNil(t, md.SeriesIndex)
	assert.Equal(t, 1.0, *md.SeriesIndex)
	require.NotNil(t, md.Cover)
	assert.Equal(t, "image/png", md.Cover.MIMEType)
	assert.NotEmpty(t, md.Cover.Data)
}

func TestExtract_EPUB_MinimalMetadata_FallsBackToFilename(t *testing.T) {
	// EPUB with no title — expect filename to supply it.
	path := buildEPUB(t, epubSpec{
		Filename: "Some Author - Some Title.epub",
		OPF: `<?xml version="1.0"?>
<package xmlns="http://www.idpf.org/2007/opf" version="3.0">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/"></metadata>
  <manifest></manifest>
</package>`,
	})
	md, err := Extract(path)
	require.NoError(t, err)
	assert.Equal(t, "Some Title", md.Title)
	require.Len(t, md.Authors, 1)
	assert.Equal(t, "Some Author", md.Authors[0])
}

func TestExtract_CorruptEPUB_FallsBack(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "Author - Title.epub")
	require.NoError(t, os.WriteFile(path, []byte("not an epub"), 0o644))

	md, err := Extract(path)
	require.NoError(t, err) // extractor swallows parse errors
	assert.Equal(t, "Title", md.Title)
	require.Len(t, md.Authors, 1)
	assert.Equal(t, "Author", md.Authors[0])
}

func TestExtract_UnsupportedFormat(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "x.txt")
	require.NoError(t, os.WriteFile(path, []byte("x"), 0o644))
	_, err := Extract(path)
	require.Error(t, err)
}

// --- test helpers ---

type epubSpec struct {
	Filename string // defaults to "book.epub"
	OPF      string
	Cover    []byte
}

// buildEPUB writes a minimal valid EPUB zip to a temp dir and returns its
// path. It contains mimetype, META-INF/container.xml, content.opf, and
// optionally cover.png.
func buildEPUB(t *testing.T, spec epubSpec) string {
	t.Helper()
	if spec.Filename == "" {
		spec.Filename = "book.epub"
	}
	path := filepath.Join(t.TempDir(), spec.Filename)
	f, err := os.Create(path)
	require.NoError(t, err)
	defer f.Close()

	zw := zip.NewWriter(f)
	// mimetype (stored, not deflated, per epub spec — here we don't care).
	mt, err := zw.Create("mimetype")
	require.NoError(t, err)
	_, err = mt.Write([]byte("application/epub+zip"))
	require.NoError(t, err)

	container, err := zw.Create("META-INF/container.xml")
	require.NoError(t, err)
	_, err = container.Write([]byte(`<?xml version="1.0"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <rootfiles><rootfile full-path="content.opf" media-type="application/oebps-package+xml"/></rootfiles>
</container>`))
	require.NoError(t, err)

	opf, err := zw.Create("content.opf")
	require.NoError(t, err)
	_, err = opf.Write([]byte(spec.OPF))
	require.NoError(t, err)

	if len(spec.Cover) > 0 {
		cw, err := zw.Create("cover.png")
		require.NoError(t, err)
		_, err = cw.Write(spec.Cover)
		require.NoError(t, err)
	}
	require.NoError(t, zw.Close())
	return path
}
