package codegraph

import (
	"go/parser"
	"go/token"
	"path"
	"path/filepath"
	"strconv"
	"strings"
)

// FileImports are the imports of one Go file that belong to its module.
type FileImports struct {
	File    string   // path relative to the module root, slash-separated
	Package string   // import path of the file's package
	Imports []string // in-module import paths, in file order
	Lines   []int    // 1-based line of each import
}

// ModuleImports reads the imports of the given Go files of the module at root.
// Only imports of the module's own packages are kept: a dependency outside the
// module is not part of the module's graph. A file that cannot be parsed is
// left out.
func ModuleImports(root string, files []string) ([]FileImports, error) {
	module, err := modulePath(root)
	if err != nil {
		return nil, err
	}
	fset := token.NewFileSet()
	var out []FileImports
	for _, rel := range files {
		f, err := parser.ParseFile(fset, filepath.Join(root, rel), nil, parser.ImportsOnly)
		if err != nil {
			continue
		}
		slashRel := filepath.ToSlash(rel)
		fi := FileImports{File: slashRel, Package: path.Join(module, path.Dir(slashRel))}
		for _, s := range f.Imports {
			p, err := strconv.Unquote(s.Path.Value)
			if err != nil {
				continue
			}
			if p == module || strings.HasPrefix(p, module+"/") {
				fi.Imports = append(fi.Imports, p)
				fi.Lines = append(fi.Lines, fset.Position(s.Pos()).Line)
			}
		}
		out = append(out, fi)
	}
	return out, nil
}
