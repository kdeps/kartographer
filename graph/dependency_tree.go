package graph

func (dg *DependencyGraph) ListDependencyTree(node string) {
	visited := make(map[string]bool)
	dg.ListDependenciesRecursive(node, []string{}, visited)
}
