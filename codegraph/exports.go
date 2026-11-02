package codegraph

import (
	"fmt"
	"html"
	"io"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// WriteCypher writes the graph as Cypher CREATE statements, for Neo4j or
// FalkorDB. Every symbol is a Symbol node; every call is a CALLS relationship.
func WriteCypher(w io.Writer, ix *Index) error {
	names := sortedNames(ix)
	ids := map[string]int{}
	for i, n := range names {
		ids[n] = i
		s := ix.Symbols[n]
		if _, err := fmt.Fprintf(w, "CREATE (n%d:Symbol {name: %q, file: %q, line: %d, language: %q});\n",
			i, s.Name, s.File, s.Line, s.Language); err != nil {
			return err
		}
	}
	for _, e := range ix.Edges() {
		if _, err := fmt.Fprintf(w, "MATCH (a:Symbol {name: %q}), (b:Symbol {name: %q}) CREATE (a)-[:CALLS {confidence: %q}]->(b);\n",
			e[0], e[1], ix.Confidence(e[0], e[1])); err != nil {
			return err
		}
	}
	return nil
}

// WriteObsidian writes one markdown note per symbol into dir. Each note links
// to the symbols it calls and the symbols that call it, so Obsidian draws the
// graph.
func WriteObsidian(dir string, ix *Index) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	for _, n := range sortedNames(ix) {
		s := ix.Symbols[n]
		var b strings.Builder
		fmt.Fprintf(&b, "# %s\n\n%s:%d\n\nlanguage: %s\n\n## Calls\n", s.Name, s.File, s.Line, s.Language)
		for _, c := range ix.deps.NodeDependencies[n] {
			fmt.Fprintf(&b, "- [[%s]]\n", noteName(c))
		}
		b.WriteString("\n## Called by\n")
		for _, c := range ix.callers[n] {
			fmt.Fprintf(&b, "- [[%s]]\n", noteName(c))
		}
		if err := os.WriteFile(filepath.Join(dir, noteName(n)+".md"), []byte(b.String()), 0o600); err != nil {
			return err
		}
	}
	return nil
}

// WriteWiki writes a markdown site into dir: an index of the files, and one
// page per file listing its symbols.
func WriteWiki(dir string, ix *Index) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	byFile := map[string][]Symbol{}
	for _, n := range sortedNames(ix) {
		s := ix.Symbols[n]
		byFile[s.File] = append(byFile[s.File], s)
	}
	files := make([]string, 0, len(byFile))
	for f := range byFile {
		files = append(files, f)
	}
	sort.Strings(files)
	var index strings.Builder
	fmt.Fprintf(&index, "# %s\n\n", ix.Module)
	for _, f := range files {
		page := fileSlug(f) + ".md"
		fmt.Fprintf(&index, "- [%s](%s)\n", f, page)
		var b strings.Builder
		fmt.Fprintf(&b, "# %s\n\n", f)
		for _, s := range byFile[f] {
			fmt.Fprintf(&b, "- %s (line %d)\n", s.Name, s.Line)
		}
		if err := os.WriteFile(filepath.Join(dir, page), []byte(b.String()), 0o600); err != nil {
			return err
		}
	}
	return os.WriteFile(filepath.Join(dir, "index.md"), []byte(index.String()), 0o600)
}

// WriteSVG draws the symbols on a circle with a line for each call.
func WriteSVG(w io.Writer, ix *Index) error {
	pos := circleLayout(ix)
	var b strings.Builder
	b.WriteString(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="-600 -600 1200 1200" width="1200" height="1200">` + "\n")
	for _, e := range ix.Edges() {
		a, c := pos[e[0]], pos[e[1]]
		fmt.Fprintf(&b, `<line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="#999"/>`+"\n", a[0], a[1], c[0], c[1])
	}
	for _, n := range sortedNames(ix) {
		p := pos[n]
		fmt.Fprintf(&b, `<circle cx="%.1f" cy="%.1f" r="5" fill="#2563eb"/>`+"\n", p[0], p[1])
		fmt.Fprintf(&b, `<text x="%.1f" y="%.1f" font-size="10">%s</text>`+"\n", p[0]+8, p[1]+4, html.EscapeString(n))
	}
	b.WriteString("</svg>\n")
	_, err := io.WriteString(w, b.String())
	return err
}

// WriteHTML writes a self-contained page with the SVG drawing and the list of
// symbols and their calls.
func WriteHTML(w io.Writer, ix *Index) error {
	var svg strings.Builder
	if err := WriteSVG(&svg, ix); err != nil {
		return err
	}
	var list strings.Builder
	for _, n := range sortedNames(ix) {
		s := ix.Symbols[n]
		fmt.Fprintf(&list, "<li><b>%s</b> %s:%d</li>\n", html.EscapeString(n), html.EscapeString(s.File), s.Line)
	}
	page := fmt.Sprintf(`<!doctype html>
<html><head><meta charset="utf-8"><title>%s</title>
<style>body{font-family:sans-serif;margin:16px}svg{max-width:100%%;height:auto}</style></head>
<body><h1>%s</h1>%s<h2>Symbols</h2><ul>%s</ul></body></html>
`, html.EscapeString(ix.Module), html.EscapeString(ix.Module), svg.String(), list.String())
	_, err := io.WriteString(w, page)
	return err
}

// circleLayout places the symbols evenly on a circle, in name order.
func circleLayout(ix *Index) map[string][2]float64 {
	names := sortedNames(ix)
	out := make(map[string][2]float64, len(names))
	for i, n := range names {
		a := 2 * math.Pi * float64(i) / float64(len(names))
		out[n] = [2]float64{500 * math.Cos(a), 500 * math.Sin(a)}
	}
	return out
}

func sortedNames(ix *Index) []string {
	names := make([]string, 0, len(ix.Symbols))
	for n := range ix.Symbols {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// noteName turns a symbol name into a file name without path separators.
func noteName(n string) string {
	return strings.NewReplacer("/", "_", ":", "_", "\\", "_").Replace(n)
}

// fileSlug turns a file path into a page name.
func fileSlug(f string) string {
	return strings.NewReplacer("/", "_", ".", "_").Replace(f)
}
