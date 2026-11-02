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
	"github.com/kdeps/kartographer/lsp"
	"github.com/kdeps/kartographer/notes"
)

// Symbol is one function or method in the module.
type Symbol struct {
	Name     string // full name, e.g. "example.com/m/pkg.(*T).Method" or "src/app.py::Login.run"
	File     string // path relative to the module root
	Line     int
	Language string // "go", or the registry id of a language server
	IsTest   bool   // defined in a test file
}

// Index is the code graph of one module.
type Index struct {
	Root    string // absolute module root
	Module  string
	Symbols map[string]Symbol
	// Missing lists languages found in the module whose language server is
	// not installed, so their symbols are absent.
	Missing []string
	deps    *graph.DependencyGraph
	callers map[string][]string
	refs    map[string][]Reference
	// inferred holds the call edges VTA resolved through an interface method,
	// keyed by caller then callee. Every other edge is extracted from a direct
	// call site or a language server reference.
	inferred map[string]map[string]bool
	// Notes is the module's local note store, used by the memory tools.
	Notes *notes.Store
}

// Build indexes every function in dir. Go modules are analysed with SSA and
// VTA. Other languages found under dir are analysed by the language server
// that registry.yaml names for them, when one is installed.
func Build(dir string) (*Index, error) {
	if !configured {
		return nil, ErrNotConfigured
	}
	root, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(root); err != nil {
		return nil, err
	}
	idx := &Index{Root: root, Symbols: map[string]Symbol{}}
	deps := map[string][]string{}

	if _, err := os.Stat(filepath.Join(root, "go.mod")); err == nil {
		if err := indexGo(root, idx, deps); err != nil {
			return nil, err
		}
	}
	if err := indexServers(root, idx, deps); err != nil {
		return nil, err
	}
	if idx.Module == "" {
		idx.Module = filepath.Base(root)
	}
	if err := idx.attachNotes(); err != nil {
		return nil, err
	}
	idx.setDeps(deps)
	return idx, nil
}

// attachNotes opens the module's note store.
func (ix *Index) attachNotes() error {
	path, err := notes.DefaultPath(ix.Module)
	if err != nil {
		return err
	}
	ix.Notes = notes.Open(path)
	return nil
}

// setDeps installs the call edges and derives the reverse map.
func (ix *Index) setDeps(deps map[string][]string) {
	ix.deps = graph.NewDependencyGraph(afero.NewMemMapFs(), log.Default(), deps)
	ix.callers = ix.deps.InvertDependencies()
}

func indexGo(root string, idx *Index, deps map[string][]string) error {
	cfg := &packages.Config{Mode: packages.LoadAllSyntax, Dir: root, Tests: true}
	pkgs, err := packages.Load(cfg, "./...")
	if err != nil {
		return err
	}
	if packages.PrintErrors(pkgs) > 0 {
		return errors.New("the packages do not build")
	}
	module, err := modulePath(root)
	if err != nil {
		return err
	}
	idx.Module = module
	prog, _ := ssautil.AllPackages(pkgs, ssa.InstantiateGenerics)
	prog.Build()
	fns := ssautil.AllFunctions(prog)
	cg := vta.CallGraph(fns, cha.CallGraph(prog))
	cg.DeleteSyntheticNodes()
	idx.refs = referencesIn(fns, root, module, prog.Fset)

	for fn := range fns {
		if !inModule(fn, module) || !fn.Pos().IsValid() {
			continue
		}
		name := displayName(fn)
		p := prog.Fset.Position(fn.Pos())
		if !strings.HasPrefix(p.Filename, root+string(filepath.Separator)) {
			continue // generated code, e.g. the test main in the build cache
		}
		rel := relPath(root, p.Filename)
		idx.Symbols[name] = Symbol{
			Name:     name,
			File:     rel,
			Line:     p.Line,
			Language: "go",
			IsTest:   strings.HasSuffix(rel, "_test.go"),
		}
		callees, inferred := calleesOf(cg, fn, module)
		deps[name] = callees
		if len(inferred) > 0 {
			if idx.inferred == nil {
				idx.inferred = map[string]map[string]bool{}
			}
			idx.inferred[name] = inferred
		}
	}
	return nil
}

// indexServers runs the registry's language server for each non-Go language
// found under root.
func indexServers(root string, idx *Index, deps map[string][]string) error {
	reg, err := lsp.LoadRegistry()
	if err != nil {
		return err
	}
	byLang := map[string][]string{}
	var order []string
	for _, rel := range sourceFiles(root) {
		lang, ok := reg.ForFile(rel)
		if !ok || lang.ID == "go" {
			continue
		}
		if _, seen := byLang[lang.ID]; !seen {
			order = append(order, lang.ID)
		}
		byLang[lang.ID] = append(byLang[lang.ID], rel)
	}
	for _, id := range order {
		lang := langByID(reg, id)
		server := lang.Server()
		if server == nil {
			idx.Missing = append(idx.Missing, fmt.Sprintf("%s: none of %v is installed", id, serverNames(lang)))
			continue
		}
		res, err := lsp.Analyze(root, id, server, byLang[id])
		if err != nil {
			return fmt.Errorf("%s: %w", id, err)
		}
		for _, f := range res.Funcs {
			idx.Symbols[f.Key] = Symbol{Name: f.Key, File: f.File, Line: f.Line, Language: id, IsTest: isTestFile(f.File)}
			deps[f.Key] = res.Calls[f.Key]
		}
		mergeRefs(idx, res.Refs)
	}
	return nil
}

// mergeRefs records the non-call uses a language server found.
func mergeRefs(idx *Index, refs []lsp.Ref) {
	if idx.refs == nil {
		idx.refs = map[string][]Reference{}
	}
	for _, r := range refs {
		idx.refs[r.Target] = append(idx.refs[r.Target], Reference{Symbol: r.Target, From: r.From, File: r.File, Line: r.Line})
	}
}

// sourceFiles lists the files under root, skipping hidden and dependency directories.
func sourceFiles(root string) []string {
	var out []string
	_ = filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if p != root && (strings.HasPrefix(d.Name(), ".") || skipDirs[d.Name()]) {
				return filepath.SkipDir
			}
			return nil
		}
		rel, err := filepath.Rel(root, p)
		if err == nil {
			out = append(out, filepath.ToSlash(rel))
		}
		return nil
	})
	sort.Strings(out)
	return out
}

var skipDirs = map[string]bool{ //nolint:gochecknoglobals // fixed list
	"node_modules": true, "vendor": true, "target": true, "build": true,
	"dist": true, "__pycache__": true,
}

func isTestFile(rel string) bool {
	base := filepath.Base(rel)
	return strings.Contains(base, "_test.") || strings.Contains(base, ".test.") ||
		strings.Contains(base, ".spec.") || strings.HasPrefix(base, "test_") ||
		strings.Contains(filepath.ToSlash(rel), "/tests/") || strings.HasPrefix(rel, "tests/")
}

func langByID(reg lsp.Registry, id string) lsp.Language {
	for _, l := range reg.Languages {
		if l.ID == id {
			return l
		}
	}
	return lsp.Language{}
}

func serverNames(l lsp.Language) []string {
	var out []string
	for _, s := range l.Servers {
		out = append(out, s[0])
	}
	return out
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

// Confidence reports how the call edge from caller to callee was found.
// "INFERRED" means VTA resolved an interface method call through type
// propagation, so the target is an approximation. "EXTRACTED" means a direct
// call site or a language server reference.
func (ix *Index) Confidence(caller, callee string) string {
	if ix.inferred[caller][callee] {
		return "INFERRED"
	}
	return "EXTRACTED"
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
		if strings.HasSuffix(name, "."+q) || strings.HasSuffix(name, "/"+q) || strings.HasSuffix(name, "::"+q) {
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
// inferred names the callees that were reached only through interface method
// calls, which VTA resolves by type propagation rather than a direct call.
func calleesOf(cg *callgraph.Graph, fn *ssa.Function, module string) ([]string, map[string]bool) {
	node := cg.Nodes[fn]
	if node == nil {
		return nil, nil
	}
	direct := map[string]bool{}
	seen := map[string]bool{}
	var out []string
	for _, e := range node.Out {
		if e.Site == nil || e.Callee.Func == nil || !inModule(e.Callee.Func, module) {
			continue
		}
		name := displayName(e.Callee.Func)
		if !e.Site.Common().IsInvoke() {
			direct[name] = true
		}
		if !seen[name] {
			seen[name] = true
			out = append(out, name)
		}
	}
	sort.Strings(out)
	inferred := map[string]bool{}
	for _, name := range out {
		if !direct[name] {
			inferred[name] = true
		}
	}
	return out, inferred
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

// names are the symbol names, in order.
func names(s []Symbol) []string {
	out := make([]string, 0, len(s))
	for _, x := range s {
		out = append(out, x.Name)
	}
	return out
}
