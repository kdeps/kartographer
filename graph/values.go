package graph

import "strings"

// Direct returns the direct dependencies of node.
func (dg *DependencyGraph) Direct(node string) []string {
	return append([]string{}, dg.NodeDependencies[node]...)
}

// Recursive returns the dependencies reached from node, depth first. A node
// reached again through another path is listed again, but its own
// dependencies are walked once.
func (dg *DependencyGraph) Recursive(node string) []string {
	out := []string{}
	visited := map[string]bool{}
	var walk func(string)
	walk = func(n string) {
		if visited[n] {
			return
		}
		visited[n] = true
		for _, d := range dg.NodeDependencies[n] {
			out = append(out, d)
			walk(d)
		}
	}
	walk(node)
	return out
}

// Reverse returns the nodes that depend on node directly.
func (dg *DependencyGraph) Reverse(node string) []string {
	return append([]string{}, dg.InvertDependencies()[node]...)
}

// Stack returns the nodes reached from node, dependencies before dependents,
// with node last.
func (dg *DependencyGraph) Stack(node string) []string {
	out := []string{}
	visited := map[string]bool{}
	var walk func(string)
	walk = func(n string) {
		if visited[n] {
			return
		}
		visited[n] = true
		for _, d := range dg.NodeDependencies[n] {
			walk(d)
		}
		out = append(out, n)
	}
	walk(node)
	return out
}

// Paths returns each distinct forward path from node, as the sequence of nodes.
// A node is entered once; a path already reported is not walked again.
func (dg *DependencyGraph) Paths(node string) [][]string {
	out := [][]string{}
	entered := map[string]bool{}
	seen := map[string]bool{}
	var current []string
	var walk func(string)
	walk = func(n string) {
		if entered[n] {
			return
		}
		entered[n] = true
		current = append(current, n)
		key := strings.Join(current, " -> ")
		if !seen[key] {
			seen[key] = true
			out = append(out, append([]string{}, current...))
			for _, d := range dg.NodeDependencies[n] {
				walk(d)
			}
		}
		current = current[:len(current)-1]
	}
	walk(node)
	return out
}
