package search

import (
	"archive/zip"
	"encoding/xml"
	"errors"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/ledongthuc/pdf"
)

// ReadText is a file's text for indexing. PDFs and
// ZIP-based office formats are extracted to plain text (see below); every
// other indexable extension is read as-is, since prior extension gating
// already limits this to text-shaped formats.
func ReadText(p string) (string, error) {
	switch strings.ToLower(filepath.Ext(p)) {
	case ".pdf":
		return extractPDFText(p)
	case ".docx":
		return extractZippedXMLText(p, "word/document.xml")
	case ".xlsx":
		return extractZippedXMLText(p, "xl/sharedStrings.xml")
	case ".odt", ".ods", ".odp":
		return extractZippedXMLText(p, "content.xml")
	case ".pptx":
		return extractPptxText(p)
	default:
		content, err := os.ReadFile(p)
		if err != nil {
			return "", err
		}
		return string(content), nil
	}
}

// extractZippedXMLText opens a ZIP-based document and extracts the text
// content of the given member file(s) (docx's word/document.xml, xlsx's
// shared-string table, or an ODF package's single content.xml), stripping
// XML markup down to the visible text nodes. Missing members are skipped
// rather than erroring, since a genuinely empty or unusually-structured
// document should still index as "no text found", not fail the whole file.
func extractZippedXMLText(p string, members ...string) (string, error) {
	r, err := zip.OpenReader(p)
	if err != nil {
		return "", err
	}
	defer func() { _ = r.Close() }()

	var sb strings.Builder
	for _, f := range r.File {
		if !slices.Contains(members, f.Name) {
			continue
		}
		if text, terr := extractXMLText(f); terr == nil {
			sb.WriteString(text)
			sb.WriteString("\n")
		}
	}
	return sb.String(), nil
}

// extractPptxText extracts every slide's text in a PowerPoint package.
// Slides are numbered member files (ppt/slides/slide1.xml, slide2.xml, ...)
// rather than one fixed path like docx/xlsx/odf, so this walks the archive
// instead of using extractZippedXMLText's fixed member list.
func extractPptxText(p string) (string, error) {
	r, err := zip.OpenReader(p)
	if err != nil {
		return "", err
	}
	defer func() { _ = r.Close() }()

	var sb strings.Builder
	for _, f := range r.File {
		if !strings.HasPrefix(f.Name, "ppt/slides/slide") || !strings.HasSuffix(f.Name, ".xml") {
			continue
		}
		if text, terr := extractXMLText(f); terr == nil {
			sb.WriteString(text)
			sb.WriteString("\n")
		}
	}
	return sb.String(), nil
}

// extractXMLText reads a zipped XML member and concatenates its character
// data nodes, discarding markup -- good enough for search indexing, not a
// faithful document reconstruction.
func extractXMLText(f *zip.File) (string, error) {
	rc, err := f.Open()
	if err != nil {
		return "", err
	}
	defer func() { _ = rc.Close() }()

	dec := xml.NewDecoder(rc)
	var sb strings.Builder
	for {
		tok, terr := dec.Token()
		if errors.Is(terr, io.EOF) {
			break
		}
		if terr != nil {
			break
		}
		if cd, ok := tok.(xml.CharData); ok {
			sb.Write(cd)
			sb.WriteByte(' ')
		}
	}
	return sb.String(), nil
}

// extractPDFText extracts plain text from a PDF using a pure-Go parser (no
// pdftotext/pdfcpu binary required), so indexing works out of the box.
// Unparsable pages are skipped rather than failing the whole file -- a
// best-effort partial index beats no index for a PDF with one bad page.
func extractPDFText(p string) (string, error) {
	f, err := os.Open(p)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()

	info, err := f.Stat()
	if err != nil {
		return "", err
	}

	reader, err := pdf.NewReader(f, info.Size())
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	fonts := make(map[string]*pdf.Font)
	for i := 1; i <= reader.NumPage(); i++ {
		page := reader.Page(i)
		for _, name := range page.Fonts() {
			if _, ok := fonts[name]; !ok {
				fnt := page.Font(name)
				fonts[name] = &fnt
			}
		}
		text, perr := page.GetPlainText(fonts)
		if perr != nil {
			continue
		}
		sb.WriteString(text)
		sb.WriteString("\n")
	}
	return sb.String(), nil
}
