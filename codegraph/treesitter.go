package codegraph

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/kdeps/kartographer/treesitter"
)

// indexTreeSitter reads the functions and calls of files with a tree-sitter
// grammar, for a language whose language server is not installed. A call is
// matched to every function of that name in the module, so each such edge is
// an approximation and is tagged INFERRED.
func indexTreeSitter(root, id string, files []string, idx *Index, deps map[string][]string) error {
	type caller struct {
		key   string
		calls []treesitter.Call
	}
	byName := map[string][]string{}
	var callers []caller
	for _, rel := range files {
		src, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			return err
		}
		funcs, err := treesitter.Extract(rel, src)
		if err != nil {
			return fmt.Errorf("%s: %w", rel, err)
		}
		for _, f := range funcs {
			key := rel + "::" + f.Name
			if _, taken := idx.Symbols[key]; taken {
				key = fmt.Sprintf("%s@%d", key, f.Line)
			}
			idx.Symbols[key] = Symbol{Name: key, File: rel, Line: f.Line, Language: id, IsTest: isTestFile(rel)}
			byName[f.Name] = append(byName[f.Name], key)
			callers = append(callers, caller{key: key, calls: f.Calls})
		}
	}
	for _, c := range callers {
		seen := map[string]bool{}
		for _, call := range c.calls {
			for _, target := range byName[call.Callee] {
				if target == c.key || seen[target] {
					continue
				}
				seen[target] = true
				deps[c.key] = append(deps[c.key], target)
				if idx.inferred == nil {
					idx.inferred = map[string]map[string]bool{}
				}
				if idx.inferred[c.key] == nil {
					idx.inferred[c.key] = map[string]bool{}
				}
				idx.inferred[c.key][target] = true
			}
		}
		sort.Strings(deps[c.key])
	}
	return nil
}
