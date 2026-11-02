package codegraph

import (
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"golang.org/x/tools/go/ssa"
)

// errNotGo is returned for files that are not Go source.
var errNotGo = errors.New("imports_exports supports Go files only")

// Reference is a use of a symbol that is not a direct call: the function is
// passed, stored, or captured as a value.
type Reference struct {
	Symbol string `json:"symbol"` // the referenced symbol
	From   string `json:"from"`   // the symbol whose body makes the reference
	File   string `json:"file"`
	Line   int    `json:"line"`
}

// referencesIn records, for each module function fn, the module functions it
// uses as values. A call whose callee is that function is not a reference.
func referencesIn(fns map[*ssa.Function]bool, root string, mods modules, fset *token.FileSet) map[string][]Reference {
	out := map[string][]Reference{}
	for fn := range fns {
		if !inModule(fn, mods) {
			continue
		}
		from := displayName(fn)
		for _, b := range fn.Blocks {
			for _, ins := range b.Instrs {
				callee := calleeOf(ins)
				for _, op := range ins.Operands(nil) {
					target, ok := (*op).(*ssa.Function)
					if !ok || (callee != nil && target == callee) || !inModule(target, mods) || !target.Pos().IsValid() {
						continue
					}
					p := fset.Position(ins.Pos())
					if !strings.HasPrefix(p.Filename, root+string(filepath.Separator)) {
						continue
					}
					name := displayName(target)
					out[name] = append(out[name], Reference{Symbol: name, From: from, File: relPath(root, p.Filename), Line: p.Line})
				}
			}
		}
	}
	return out
}

// calleeOf is the function a call instruction calls directly, or nil.
func calleeOf(ins ssa.Instruction) ssa.Value {
	if ci, ok := ins.(ssa.CallInstruction); ok {
		return ci.Common().Value
	}
	return nil
}

// References returns the non-call uses of a symbol, sorted by location.
func (ix *Index) References(symbol string, limit int) ([]Reference, error) {
	name, err := ix.resolve(symbol)
	if err != nil {
		return nil, err
	}
	refs := append([]Reference{}, ix.refs[name]...)
	sort.Slice(refs, func(i, j int) bool {
		if refs[i].File != refs[j].File {
			return refs[i].File < refs[j].File
		}
		return refs[i].Line < refs[j].Line
	})
	if limit > 0 && len(refs) > limit {
		refs = refs[:limit]
	}
	return refs, nil
}

// ImportsExports lists the import paths and exported top-level names of a Go
// file under the module root. Other languages are not analysed.
func (ix *Index) ImportsExports(file string) (imports, exports []string, err error) {
	if filepath.Ext(file) != ".go" {
		return nil, nil, errNotGo
	}
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filepath.Join(ix.Root, file), nil, parser.ParseComments)
	if err != nil {
		return nil, nil, err
	}
	for _, imp := range f.Imports {
		if path, err := strconv.Unquote(imp.Path.Value); err == nil {
			imports = append(imports, path)
		}
	}
	for _, d := range f.Decls {
		exports = append(exports, exportedNames(d)...)
	}
	sort.Strings(imports)
	sort.Strings(exports)
	return imports, exports, nil
}

// exportedNames are the exported functions (not methods), types, variables
// and constants a declaration introduces.
func exportedNames(d ast.Decl) []string {
	switch d := d.(type) {
	case *ast.FuncDecl:
		if d.Recv == nil && isExported(d.Name.Name) {
			return []string{d.Name.Name}
		}
	case *ast.GenDecl:
		var out []string
		for _, spec := range d.Specs {
			switch s := spec.(type) {
			case *ast.TypeSpec:
				if isExported(s.Name.Name) {
					out = append(out, s.Name.Name)
				}
			case *ast.ValueSpec:
				for _, n := range s.Names {
					if isExported(n.Name) {
						out = append(out, n.Name)
					}
				}
			}
		}
		return out
	}
	return nil
}

func isExported(name string) bool {
	r := []rune(name)
	return len(r) > 0 && unicode.IsUpper(r[0])
}
