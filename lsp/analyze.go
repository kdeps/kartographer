package lsp

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// Func is a function or method a language server reports.
type Func struct {
	Key  string // file path and qualified name, e.g. "src/app.py::Login.run"
	File string // path relative to the root
	Name string // qualified name within the file
	Line int    // 1-based line of the name
	End  int    // 1-based line where the body ends
	Col  int    // 0-based column of the name
}

// Ref is a use of a function that is not a call: a function value, a callback,
// or a reference from another initialiser. From is the function whose body
// makes the use.
type Ref struct {
	Target string
	From   string
	File   string
	Line   int
}

// Result is what a server reported for a set of files.
type Result struct {
	Funcs []Func
	// Calls maps a function's key to the keys of the functions it calls.
	// Calls to functions outside the files are dropped.
	Calls map[string][]string
	// NoCallHierarchy is set when the server does not answer call hierarchy
	// requests. Funcs are still reported; Calls is empty.
	NoCallHierarchy bool
	// Refs are the uses of functions that are not call edges, from the
	// server's references answers.
	Refs []Ref
}

const (
	kindMethod      = 6
	kindConstructor = 9
	kindFunction    = 12
)

type position struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

type span struct {
	Start position `json:"start"`
	End   position `json:"end"`
}

type docSymbol struct {
	Name           string      `json:"name"`
	Kind           int         `json:"kind"`
	Range          span        `json:"range"`
	SelectionRange span        `json:"selectionRange"`
	Children       []docSymbol `json:"children"`
}

type callItem struct {
	Name           string `json:"name"`
	Kind           int    `json:"kind"`
	URI            string `json:"uri"`
	SelectionRange span   `json:"selectionRange"`
}

// Analyze runs server (a command and its arguments) over files under root and
// returns their functions and the calls between them. Files are relative to root.
func Analyze(root, langID string, server []string, files []string) (Result, error) {
	c, err := Start(server[0], server[1:]...)
	if err != nil {
		return Result{}, err
	}
	defer c.Close()

	rootURI := fileURI(root)
	if err := c.Call("initialize", map[string]any{
		"processId": nil,
		"rootUri":   rootURI,
		"capabilities": map[string]any{"textDocument": map[string]any{
			"documentSymbol": map[string]any{"hierarchicalDocumentSymbolSupport": true},
			"callHierarchy":  map[string]any{"dynamicRegistration": false},
		}},
	}, nil); err != nil {
		return Result{}, err
	}
	if err := c.Notify("initialized", map[string]any{}); err != nil {
		return Result{}, err
	}

	var res Result
	byLine := map[string]string{} // "uri:line" -> key
	for _, rel := range files {
		abs := filepath.Join(root, rel)
		text, err := os.ReadFile(abs)
		if err != nil {
			return Result{}, err
		}
		uri := fileURI(abs)
		if err := c.Notify("textDocument/didOpen", map[string]any{"textDocument": map[string]any{
			"uri": uri, "languageId": langID, "version": 1, "text": string(text),
		}}); err != nil {
			return Result{}, err
		}
		var syms []docSymbol
		if err := c.Call("textDocument/documentSymbol", map[string]any{
			"textDocument": map[string]any{"uri": uri},
		}, &syms); err != nil {
			return Result{}, err
		}
		for _, f := range flatten(rel, "", syms, strings.Split(string(text), "\n")) {
			res.Funcs = append(res.Funcs, f)
			byLine[fmt.Sprintf("%s:%d", uri, f.Line-1)] = f.Key
		}
	}

	res.Calls = map[string][]string{}
	for _, f := range res.Funcs {
		uri := fileURI(filepath.Join(root, f.File))
		var items []json.RawMessage
		err := c.Call("textDocument/prepareCallHierarchy", map[string]any{
			"textDocument": map[string]any{"uri": uri},
			"position":     position{Line: f.Line - 1, Character: f.Col},
		}, &items)
		if err != nil {
			res.NoCallHierarchy = true
			continue
		}
		for _, it := range items {
			// Incoming calls: some servers (clangd) answer only these. Each
			// caller gets an edge to f; the graph inverts it for callees. The
			// item goes back exactly as the server sent it: servers match it
			// by fields such as data, which a decoded copy would drop.
			var ins []struct {
				From callItem `json:"from"`
			}
			if err := c.Call("callHierarchy/incomingCalls", map[string]any{"item": it}, &ins); err != nil {
				if strings.Contains(err.Error(), "method not found") {
					res.NoCallHierarchy = true
				}
				continue
			}
			for _, in := range ins {
				key, ok := byLine[fmt.Sprintf("%s:%d", in.From.URI, in.From.SelectionRange.Start.Line)]
				if ok && key != f.Key {
					res.Calls[key] = appendUnique(res.Calls[key], f.Key)
				}
			}
		}
	}
	res.Refs = usesOf(c, root, res)
	return res, nil
}

// usesOf asks the server where each function is referenced. A call site is
// already an edge in res.Calls, so only the other uses are kept.
func usesOf(c *Client, root string, res Result) []Ref {
	relOf := map[string]string{}
	for _, f := range res.Funcs {
		relOf[fileURI(filepath.Join(root, f.File))] = f.File
	}
	var out []Ref
	for _, f := range res.Funcs {
		var locs []location
		err := c.Call("textDocument/references", map[string]any{
			"textDocument": map[string]any{"uri": fileURI(filepath.Join(root, f.File))},
			"position":     position{Line: f.Line - 1, Character: f.Col},
			"context":      map[string]any{"includeDeclaration": false},
		}, &locs)
		if err != nil {
			continue
		}
		out = append(out, referenceSites(res.Funcs, res.Calls, f.Key, relOf, locs)...)
	}
	return out
}

// location is a file position in a references answer.
type location struct {
	URI   string `json:"uri"`
	Range span   `json:"range"`
}

// referenceSites turns the locations where target is referenced into Refs.
// Each location is attributed to the innermost function whose body contains
// it. A location that is already a call edge from that function is dropped.
func referenceSites(funcs []Func, calls map[string][]string, target string, relOf map[string]string, locs []location) []Ref {
	var out []Ref
	for _, l := range locs {
		rel, ok := relOf[l.URI]
		if !ok {
			continue
		}
		line := l.Range.Start.Line + 1
		from, ok := enclosing(funcs, rel, line)
		if !ok || slices.Contains(calls[from.Key], target) {
			continue
		}
		out = append(out, Ref{Target: target, From: from.Key, File: rel, Line: line})
	}
	return out
}

// enclosing returns the innermost function in file whose body contains line.
func enclosing(funcs []Func, file string, line int) (Func, bool) {
	var best Func
	found := false
	for _, f := range funcs {
		if f.File != file || line < f.Line || line > f.End {
			continue
		}
		if !found || f.Line > best.Line {
			best, found = f, true
		}
	}
	return best, found
}

// flatten walks document symbols, keeping functions and methods with their
// qualified name (Outer.Inner). A declaration without a body, such as a C
// prototype or a TypeScript overload signature, ends in ";" and is skipped.
func flatten(rel, prefix string, syms []docSymbol, lines []string) []Func {
	var out []Func
	for _, s := range syms {
		name := s.Name
		if prefix != "" {
			name = prefix + "." + s.Name
		}
		if (s.Kind == kindFunction || s.Kind == kindMethod || s.Kind == kindConstructor) && !isDeclaration(s.Range, lines) {
			out = append(out, Func{
				Key:  rel + "::" + name,
				File: rel,
				Name: name,
				Line: s.SelectionRange.Start.Line + 1,
				End:  s.Range.End.Line + 1,
				Col:  s.SelectionRange.Start.Character,
			})
		}
		out = append(out, flatten(rel, name, s.Children, lines)...)
	}
	return out
}

// isDeclaration reports whether the symbol ends in ";" with no body. Servers
// end the range before the ";", so the text after the range is checked.
func isDeclaration(r span, lines []string) bool {
	end := r.End.Line
	if end < 0 || end >= len(lines) {
		return false
	}
	line := lines[end]
	rest := ""
	if r.End.Character < len(line) {
		rest = line[r.End.Character:]
	}
	return strings.HasPrefix(strings.TrimSpace(rest), ";")
}

func appendUnique(s []string, v string) []string {
	for _, x := range s {
		if x == v {
			return s
		}
	}
	return append(s, v)
}

func fileURI(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = path
	}
	return "file://" + filepath.ToSlash(abs)
}
