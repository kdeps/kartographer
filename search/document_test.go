package search

import (
	"archive/zip"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func zipWith(t *testing.T, path string, members map[string]string) {
	t.Helper()
	f, err := os.Create(path)
	require.NoError(t, err)
	defer func() { _ = f.Close() }()
	zw := zip.NewWriter(f)
	for name, content := range members {
		w, err := zw.Create(name)
		require.NoError(t, err)
		_, err = w.Write([]byte(content))
		require.NoError(t, err)
	}
	require.NoError(t, zw.Close())
}

// onePagePDF is a minimal PDF whose one page shows text, with a correct xref.
func onePagePDF(text string) []byte {
	stream := "BT /F1 12 Tf 10 100 Td (" + text + ") Tj ET"
	objs := []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /Resources << /Font << /F1 4 0 R >> >> /MediaBox [0 0 200 200] /Contents 5 0 R >>",
		"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>",
		fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(stream), stream),
	}
	var b bytes.Buffer
	b.WriteString("%PDF-1.4\n")
	offsets := make([]int, len(objs))
	for i, body := range objs {
		offsets[i] = b.Len()
		fmt.Fprintf(&b, "%d 0 obj\n%s\nendobj\n", i+1, body)
	}
	xref := b.Len()
	fmt.Fprintf(&b, "xref\n0 %d\n0000000000 65535 f \n", len(objs)+1)
	for _, off := range offsets {
		fmt.Fprintf(&b, "%010d 00000 n \n", off)
	}
	fmt.Fprintf(&b, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF", len(objs)+1, xref)
	return b.Bytes()
}

func TestReadText(t *testing.T) {
	dir := t.TempDir()
	cases := map[string]map[string]string{
		".docx": {"word/document.xml": "<w:document><w:t>DocxWord</w:t></w:document>", "word/other.xml": "<x>Skipped</x>"},
		".xlsx": {"xl/sharedStrings.xml": "<sst><si><t>XlsxWord</t></si></sst>"},
		".odt":  {"content.xml": "<office:text><text:p>OdtWord</text:p></office:text>"},
		".pptx": {"ppt/slides/slide1.xml": "<p:sld><a:t>PptxOne</a:t></p:sld>", "ppt/slides/slide2.xml": "<a:t>PptxTwo</a:t>"},
	}
	want := map[string][]string{
		".docx": {"DocxWord"}, ".xlsx": {"XlsxWord"}, ".odt": {"OdtWord"}, ".pptx": {"PptxOne", "PptxTwo"},
	}
	for ext, members := range cases {
		p := filepath.Join(dir, "doc"+ext)
		zipWith(t, p, members)
		text, err := ReadText(p)
		require.NoError(t, err, ext)
		for _, w := range want[ext] {
			assert.Contains(t, text, w, ext)
		}
		assert.NotContains(t, text, "Skipped", ext)
	}

	pdf := filepath.Join(dir, "doc.pdf")
	require.NoError(t, os.WriteFile(pdf, onePagePDF("PdfWord"), 0o600))
	text, err := ReadText(pdf)
	require.NoError(t, err)
	assert.Contains(t, text, "PdfWord")

	plain := filepath.Join(dir, "a.md")
	require.NoError(t, os.WriteFile(plain, []byte("plain words"), 0o600))
	text, err = ReadText(plain)
	require.NoError(t, err)
	assert.Equal(t, "plain words", text)

	_, err = ReadText(filepath.Join(dir, "missing.docx"))
	require.Error(t, err)
}
