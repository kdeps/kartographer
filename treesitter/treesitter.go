// Package treesitter reads the functions and call sites of a source file with a
// tree-sitter grammar. It needs no language server, so it covers a language
// whose server is not installed. A call is matched to a function by name, so
// the edges it gives are approximations.
package treesitter

import (
	"context"
	"path/filepath"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/java"
	"github.com/smacker/go-tree-sitter/javascript"
	"github.com/smacker/go-tree-sitter/python"
	"github.com/smacker/go-tree-sitter/rust"
	"github.com/smacker/go-tree-sitter/typescript/tsx"
	"github.com/smacker/go-tree-sitter/typescript/typescript"
)

// Func is a function or method declared in a file.
type Func struct {
	Name    string // the declared name
	Line    int    // 1-based line of the name
	EndLine int    // 1-based last line of the declaration
	Calls   []Call // the call sites inside its body
}

// Call is a call site: Callee is the last identifier of the called expression.
type Call struct {
	Callee string
	Line   int
}

// grammar describes one language: the node types that declare functions and
// that make calls, and the field of a call that holds the called expression.
type grammar struct {
	id        string
	lang      *sitter.Language
	funcs     map[string]bool
	calls     map[string]bool
	calleeKey string
}

var byExt = map[string]grammar{
	".py":   {id: "python", lang: python.GetLanguage(), funcs: set("function_definition"), calls: set("call"), calleeKey: "function"},
	".js":   jsGrammar("javascript", javascript.GetLanguage()),
	".jsx":  jsGrammar("javascript", javascript.GetLanguage()),
	".mjs":  jsGrammar("javascript", javascript.GetLanguage()),
	".cjs":  jsGrammar("javascript", javascript.GetLanguage()),
	".ts":   jsGrammar("typescript", typescript.GetLanguage()),
	".tsx":  jsGrammar("typescript", tsx.GetLanguage()),
	".java": {id: "java", lang: java.GetLanguage(), funcs: set("method_declaration", "constructor_declaration"), calls: set("method_invocation"), calleeKey: "name"},
	".rs":   {id: "rust", lang: rust.GetLanguage(), funcs: set("function_item"), calls: set("call_expression"), calleeKey: "function"},
}

func jsGrammar(id string, lang *sitter.Language) grammar {
	return grammar{id: id, lang: lang,
		funcs:     set("function_declaration", "method_definition"),
		calls:     set("call_expression"),
		calleeKey: "function"}
}

func set(types ...string) map[string]bool {
	out := make(map[string]bool, len(types))
	for _, t := range types {
		out[t] = true
	}
	return out
}

// Supports reports whether a file of the language with id can be read.
func Supports(id string) bool {
	for _, g := range byExt {
		if g.id == id {
			return true
		}
	}
	return false
}

// Languages returns the language id of a file path, or "" when none has a grammar.
func Language(path string) string {
	g, ok := byExt[strings.ToLower(filepath.Ext(path))]
	if !ok {
		return ""
	}
	return g.id
}

// Extract returns the functions declared in src, a file at path, each with its
// call sites. A file with no grammar gives nothing.
func Extract(path string, src []byte) ([]Func, error) {
	g, ok := byExt[strings.ToLower(filepath.Ext(path))]
	if !ok {
		return nil, nil
	}
	parser := sitter.NewParser()
	defer parser.Close()
	parser.SetLanguage(g.lang)
	tree, err := parser.ParseCtx(context.Background(), nil, src)
	if err != nil {
		return nil, err
	}
	if tree == nil {
		return nil, nil
	}
	defer tree.Close()
	var out []Func
	g.walk(tree.RootNode(), src, -1, &out)
	return out, nil
}

// walk visits the named nodes under n. cur is the function whose body n is in,
// or -1 outside any function. A call is recorded against cur.
func (g grammar) walk(n *sitter.Node, src []byte, cur int, out *[]Func) {
	if g.funcs[n.Type()] {
		if name := n.ChildByFieldName("name"); name != nil {
			*out = append(*out, Func{
				Name:    name.Content(src),
				Line:    int(name.StartPoint().Row) + 1,
				EndLine: int(n.EndPoint().Row) + 1,
			})
			cur = len(*out) - 1
		}
	}
	if g.calls[n.Type()] && cur >= 0 {
		if callee := n.ChildByFieldName(g.calleeKey); callee != nil {
			(*out)[cur].Calls = append((*out)[cur].Calls, Call{
				Callee: lastIdent(callee.Content(src)),
				Line:   int(n.StartPoint().Row) + 1,
			})
		}
	}
	for i := 0; i < int(n.NamedChildCount()); i++ {
		g.walk(n.NamedChild(i), src, cur, out)
	}
}

// lastIdent is the last identifier in s: "obj.run()" gives "run".
func lastIdent(s string) string {
	i := len(s)
	for i > 0 && !isIdent(s[i-1]) {
		i--
	}
	j := i
	for j > 0 && isIdent(s[j-1]) {
		j--
	}
	return s[j:i]
}

func isIdent(c byte) bool {
	return c == '_' || c == '$' || c >= '0' && c <= '9' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z'
}
