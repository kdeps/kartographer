// Package codegraph indexes the symbols and resolved call edges of a Go module
// and answers the queries an agent asks of a code graph: where a symbol is,
// who calls it, what it calls, and which tests reach it.
package codegraph

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/afero"
	"golang.org/x/mod/modfile"
	"golang.org/x/tools/go/callgraph"
	"golang.org/x/tools/go/callgraph/cha"
	"golang.org/x/tools/go/callgraph/vta"
	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"

	"github.com/charmbracelet/log"

	"github.com/kdeps/kartographer/graph"
)

// Symbol is one function or method defined in the module.
type Symbol struct {
	Name   string // full name, e.g. "example.com/m/pkg.(*T).Method"
	File   string // path relative to the module root
	Line   int
	IsTest bool // defined in a _test.go file
}

// Index is the code graph of one module.
type Index struct {
	Module  string
	Symbols map[string]Symbol
	deps    *graph.DependencyGraph
	callers map[string][]string
}

// Build loads every package under dir (with tests) and records each
// module-internal function and the functions it calls.
func Build(dir string) (*Index, error) {
	root, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	cfg := &packages.Config{Mode: packages.LoadAllSyntax, Dir: root, Tests: true}
	pkgs, err := packages.Load(cfg, "./...")
	if err != nil {
		return nil, err
	}
	if packages.PrintErrors(pkgs) > 0 {
		return nil, errors.New("the packages do not build")
	}
	module, err := modulePath(root)
	if err != nil {
		return nil, err
	}
	prog, _ := ssautil.AllPackages(pkgs, ssa.InstantiateGenerics)
	prog.Build()
	fns := ssautil.AllFunctions(prog)
	cg := vta.CallGraph(fns, cha.CallGraph(prog))
	cg.DeleteSyntheticNodes()

	idx := &Index{Module: module, Symbols: map[string]Symbol{}}
	deps := map[string][]string{}
	for fn := range fns {
		if !inModule(fn, module) || !fn.Pos().IsValid() {
			continue
		}
		name := displayName(fn)
		p := prog.Fset.Position(fn.Pos())
		if !strings.HasPrefix(p.Filename, root+string(filepath.Separator)) {
			continue // generated code, e.g. the test main in the build cache
		}
		idx.Symbols[name] = Symbol{
			Name:   name,
			File:   relPath(root, p.Filename),
			Line:   p.Line,
			IsTest: strings.HasSuffix(p.Filename, "_test.go"),
		}
		deps[name] = calleesOf(cg, fn, module)
	}
	idx.deps = graph.NewDependencyGraph(afero.NewMemMapFs(), log.Default(), deps)
	idx.callers = idx.deps.InvertDependencies()
	return idx, nil
}

// Stats reports the number of symbols and call edges.
func (ix *Index) Stats() (symbols, edges int) {
	for _, cs := range ix.deps.NodeDependencies {
		edges += len(cs)
	}
	return len(ix.Symbols), edges
}

// Find returns the symbols whose full name contains term, sorted.
func (ix *Index) Find(term string, limit int) []Symbol {
	var out []Symbol
	for name, s := range ix.Symbols {
		if strings.Contains(name, term) {
			out = append(out, s)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return capped(out, limit)
}

// Callers are the functions that call the symbol directly.
func (ix *Index) Callers(symbol string, limit int) ([]Symbol, error) {
	name, err := ix.resolve(symbol)
	if err != nil {
		return nil, err
	}
	return ix.symbols(ix.callers[name], limit), nil
}

// Callees are the functions the symbol calls directly.
func (ix *Index) Callees(symbol string, limit int) ([]Symbol, error) {
	name, err := ix.resolve(symbol)
	if err != nil {
		return nil, err
	}
	return ix.symbols(ix.deps.NodeDependencies[name], limit), nil
}

// Neighbours are the direct callers and callees of the target symbol.
func (ix *Index) Neighbours(target string, limit int) ([]Symbol, error) {
	name, err := ix.resolve(target)
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	var names []string
	for _, n := range append(append([]string{}, ix.callers[name]...), ix.deps.NodeDependencies[name]...) {
		if !seen[n] {
			seen[n] = true
			names = append(names, n)
		}
	}
	return ix.symbols(names, limit), nil
}

// TestsFor are the test functions that reach the target, through any chain of
// callers. The target is a symbol name or a file path in the module.
func (ix *Index) TestsFor(target string, limit int) ([]Symbol, error) {
	starts := ix.fileSymbols(target)
	if len(starts) == 0 {
		name, err := ix.resolve(target)
		if err != nil {
			return nil, err
		}
		starts = []string{name}
	}
	seen := map[string]bool{}
	queue := append([]string{}, starts...)
	for _, s := range starts {
		seen[s] = true
	}
	var tests []string
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if ix.Symbols[cur].IsTest {
			tests = append(tests, cur)
		}
		for _, c := range ix.callers[cur] {
			if !seen[c] {
				seen[c] = true
				queue = append(queue, c)
			}
		}
	}
	sort.Strings(tests)
	return ix.symbols(tests, limit), nil
}

// fileSymbols are the symbols defined in a module file, when target is one.
func (ix *Index) fileSymbols(target string) []string {
	var out []string
	for name, s := range ix.Symbols {
		if s.File == target {
			out = append(out, name)
		}
	}
	sort.Strings(out)
	return out
}

// resolve maps a query to one full symbol name: an exact name, or a suffix
// such as "Type.Method" or "pkg.Func". An ambiguous suffix is an error.
func (ix *Index) resolve(q string) (string, error) {
	if _, ok := ix.Symbols[q]; ok {
		return q, nil
	}
	var hits []string
	for name := range ix.Symbols {
		if strings.HasSuffix(name, "."+q) || strings.HasSuffix(name, "/"+q) {
			hits = append(hits, name)
		}
	}
	switch len(hits) {
	case 0:
		return "", fmt.Errorf("no symbol matches %q", q)
	case 1:
		return hits[0], nil
	default:
		sort.Strings(hits)
		return "", fmt.Errorf("%q is ambiguous: %s", q, strings.Join(hits, ", "))
	}
}

func (ix *Index) symbols(names []string, limit int) []Symbol {
	out := make([]Symbol, 0, len(names))
	for _, n := range names {
		if s, ok := ix.Symbols[n]; ok {
			out = append(out, s)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return capped(out, limit)
}

func capped(s []Symbol, limit int) []Symbol {
	if limit > 0 && len(s) > limit {
		return s[:limit]
	}
	return s
}

// calleesOf are fn's resolved calls to functions in the module, by full name.
func calleesOf(cg *callgraph.Graph, fn *ssa.Function, module string) []string {
	node := cg.Nodes[fn]
	if node == nil {
		return nil
	}
	seen := map[string]bool{}
	var out []string
	for _, e := range node.Out {
		if e.Site == nil || e.Callee.Func == nil || !inModule(e.Callee.Func, module) {
			continue
		}
		name := displayName(e.Callee.Func)
		if !seen[name] {
			seen[name] = true
			out = append(out, name)
		}
	}
	sort.Strings(out)
	return out
}

// displayName is fn's full name with the receiver written as a type path:
// "(*pkg.T).M" and "(pkg.T).M" both become "pkg.T.M".
func displayName(fn *ssa.Function) string {
	return strings.NewReplacer("(*", "", "(", "", ")", "").Replace(fn.String())
}

func inModule(fn *ssa.Function, module string) bool {
	if fn.Pkg == nil || fn.Pkg.Pkg == nil {
		return false
	}
	p := fn.Pkg.Pkg.Path()
	return p == module || strings.HasPrefix(p, module+"/")
}

// modulePath reads the module directive of root/go.mod.
func modulePath(root string) (string, error) {
	raw, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		return "", err
	}
	mod, err := modfile.Parse("go.mod", raw, nil)
	if err != nil {
		return "", err
	}
	return mod.Module.Mod.Path, nil
}

func relPath(root, file string) string {
	r := strings.TrimSuffix(root, "/") + "/"
	return strings.TrimPrefix(file, r)
}
