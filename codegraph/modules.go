package codegraph

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/mod/modfile"
)

// modules are the module paths indexed together: the module at the root, or
// every module a go.work file at the root uses.
type modules []string

// contains reports whether a package path belongs to one of the modules.
func (ms modules) contains(path string) bool {
	for _, m := range ms {
		if path == m || strings.HasPrefix(path, m+"/") {
			return true
		}
	}
	return false
}

var errNoWorkModules = errors.New("go.work lists no modules")

// goPackages returns the package patterns to load under root, and the module
// paths they belong to. Without a go.work file that is the module at root;
// with one, each used module, so a call from one to another is an edge.
func goPackages(root string) ([]string, modules, error) {
	raw, err := os.ReadFile(filepath.Join(root, "go.work"))
	if os.IsNotExist(err) {
		mod, err := modulePath(root)
		if err != nil {
			return nil, nil, err
		}
		return []string{"./..."}, modules{mod}, nil
	}
	if err != nil {
		return nil, nil, err
	}
	wf, err := modfile.ParseWork("go.work", raw, nil)
	if err != nil {
		return nil, nil, err
	}
	var patterns []string
	var mods modules
	for _, u := range wf.Use {
		dir := filepath.Clean(filepath.Join(root, u.Path))
		mod, err := modulePath(dir)
		if err != nil {
			return nil, nil, err
		}
		mods = append(mods, mod)
		rel, err := filepath.Rel(root, dir)
		if err != nil {
			return nil, nil, err
		}
		if rel == "." {
			patterns = append(patterns, "./...")
		} else {
			patterns = append(patterns, "./"+filepath.ToSlash(rel)+"/...")
		}
	}
	if len(mods) == 0 {
		return nil, nil, errNoWorkModules
	}
	return patterns, mods, nil
}
