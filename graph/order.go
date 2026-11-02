package graph

import (
	"sort"
)

// CycleError reports a dependency cycle: Node is on it.
type CycleError struct {
	Node string
}

func (e *CycleError) Error() string { return "dependency cycle through " + e.Node }

// Order returns roots and every node they depend on, each dependency before
// the nodes that need it. With no roots it orders every node. Roots are taken
// in sorted order and dependencies in the order they are listed, so the result
// is the same on every call. A cycle stops it with a *CycleError naming a node
// on the cycle.
func (dg *DependencyGraph) Order(roots ...string) ([]string, error) {
	if len(roots) == 0 {
		seen := map[string]bool{}
		for n, deps := range dg.NodeDependencies {
			seen[n] = true
			for _, d := range deps {
				seen[d] = true
			}
		}
		for n := range seen {
			roots = append(roots, n)
		}
	}
	roots = append([]string{}, roots...)
	sort.Strings(roots)
	var out []string
	done := map[string]bool{}
	onPath := map[string]bool{}
	var walk func(string) error
	walk = func(n string) error {
		if onPath[n] {
			return &CycleError{Node: n}
		}
		if done[n] {
			return nil
		}
		onPath[n] = true
		for _, d := range dg.NodeDependencies[n] {
			if err := walk(d); err != nil {
				return err
			}
		}
		delete(onPath, n)
		done[n] = true
		out = append(out, n)
		return nil
	}
	for _, r := range roots {
		if err := walk(r); err != nil {
			return nil, err
		}
	}
	return out, nil
}
