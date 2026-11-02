package main

import (
	"bytes"
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kdeps/kartographer/codegraph"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// configureRules loads the test rules into the codegraph package, as the
// command does before it builds an index.
func configureRules(t *testing.T) {
	t.Helper()
	raw, err := os.ReadFile(rulesFile(t))
	require.NoError(t, err)
	require.NoError(t, codegraph.Configure(raw))
}

func TestExportSingleFileFormats(t *testing.T) {
	configureRules(t)
	var svg bytes.Buffer
	require.NoError(t, exportCommand(&svg, "svg", "", fixture, true))
	assert.Contains(t, svg.String(), "<svg")
	assert.Contains(t, svg.String(), "<line")
	var x struct{ XMLName xml.Name }
	require.NoError(t, xml.Unmarshal(svg.Bytes(), &x), "svg is well-formed xml")

	var html bytes.Buffer
	require.NoError(t, exportCommand(&html, "html", "", fixture, true))
	assert.Contains(t, html.String(), "<!doctype html>")
	assert.Contains(t, html.String(), "<svg")
	assert.Contains(t, html.String(), "example.com/fx/pkg.leaf")

	var cy bytes.Buffer
	require.NoError(t, exportCommand(&cy, "neo4j", "", fixture, true))
	assert.Contains(t, cy.String(), `CREATE (n0:Symbol {name: `)
	assert.Contains(t, cy.String(), "-[:CALLS]->")

	var fdb bytes.Buffer
	require.NoError(t, exportCommand(&fdb, "falkordb", "", fixture, true))
	assert.Equal(t, cy.String(), fdb.String(), "falkordb loads the same Cypher")

	path := filepath.Join(t.TempDir(), "g.html")
	var msg bytes.Buffer
	require.NoError(t, exportCommand(&msg, "html", path, fixture, true))
	assert.Contains(t, msg.String(), "wrote "+path)
	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(raw), "<!doctype html>")
}

func TestExportDirectoryFormats(t *testing.T) {
	configureRules(t)
	obs := filepath.Join(t.TempDir(), "vault")
	var out bytes.Buffer
	require.NoError(t, exportCommand(&out, "obsidian", obs, fixture, true))
	note, err := os.ReadFile(filepath.Join(obs, "example.com_fx_pkg.leaf.md"))
	require.NoError(t, err)
	assert.Contains(t, string(note), "# example.com/fx/pkg.leaf")
	assert.Contains(t, string(note), "## Called by")
	assert.Contains(t, string(note), "[[example.com_fx_pkg.helper]]")

	wiki := filepath.Join(t.TempDir(), "wiki")
	require.NoError(t, exportCommand(&out, "wiki", wiki, fixture, true))
	index, err := os.ReadFile(filepath.Join(wiki, "index.md"))
	require.NoError(t, err)
	assert.Contains(t, string(index), "(pkg_a_go.md)")
	page, err := os.ReadFile(filepath.Join(wiki, "pkg_a_go.md"))
	require.NoError(t, err)
	assert.Contains(t, string(page), "leaf (line 7)")
}

func TestExportErrors(t *testing.T) {
	configureRules(t)
	var out bytes.Buffer
	assert.ErrorContains(t, exportCommand(&out, "obsidian", "", fixture, true), "needs -o DIR")
	assert.ErrorContains(t, exportCommand(&out, "wiki", "", fixture, true), "needs -o DIR")
	assert.ErrorContains(t, exportCommand(&out, "pdf", "", fixture, true), "export supports")

	_, errw, code := runCLI(t, "export", "pdf", "-dir", fixture, "-no-cache")
	assert.Equal(t, 2, code)
	assert.Contains(t, errw, "export supports")
}

func TestAddFetchesPageWithFrontMatter(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/missing" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte("# Hello page"))
	}))
	defer srv.Close()

	dir := filepath.Join(t.TempDir(), "raw")
	var out bytes.Buffer
	require.NoError(t, addCommand(&out, srv.URL+"/notes/intro?x=1", dir, "Ada", "Grace"))
	saved := strings.TrimSpace(strings.TrimPrefix(out.String(), "saved "))
	raw, err := os.ReadFile(saved)
	require.NoError(t, err)
	body := string(raw)
	assert.Contains(t, body, `author: "Ada"`)
	assert.Contains(t, body, `contributor: "Grace"`)
	assert.Contains(t, body, `source: "`+srv.URL+`/notes/intro?x=1"`)
	assert.Contains(t, body, "# Hello page")
	assert.Equal(t, filepath.Dir(saved), dir)
}

func TestAddOmitsEmptyPeople(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("body"))
	}))
	defer srv.Close()
	dir := t.TempDir()
	var out bytes.Buffer
	require.NoError(t, addCommand(&out, srv.URL, dir, "", ""))
	entries, _ := os.ReadDir(dir)
	require.Len(t, entries, 1)
	raw, _ := os.ReadFile(filepath.Join(dir, entries[0].Name()))
	assert.NotContains(t, string(raw), "author:")
	assert.NotContains(t, string(raw), "contributor:")
}

func TestAddErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer srv.Close()
	var out bytes.Buffer
	assert.ErrorContains(t, addCommand(&out, "file:///etc/passwd", t.TempDir(), "", ""), "http or https")
	assert.ErrorContains(t, addCommand(&out, srv.URL+"/missing", t.TempDir(), "", ""), "404")

	_, errw, code := runCLI(t, "add", "-dir", fixture, "-no-cache")
	assert.Equal(t, 2, code)
	assert.Contains(t, errw, "add needs a URL")
}

func TestAddFileNameFromURL(t *testing.T) {
	assert.Equal(t, "example.com_a_b.md", addFileName("https://example.com/a/b"))
	assert.Equal(t, "page.md", addFileName("https://"))
}
