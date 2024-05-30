package graph

func (dg *DependencyGraph) ListDependenciesRecursive(node string, path []string, visited map[string]bool) {
	if visited[node] {
		return
	}
	visited[node] = true
	path = append(path, node)
	deps, exists := dg.NodeDependencies[node]
	if !exists || len(deps) == 0 {
		dg.PrintDependencyPath(dg.ConstructDependencyPath(path, "reverse"))
	} else {
		for _, dep := range deps {
			dg.ListDependenciesRecursive(dep, path, visited)
		}
	}
	visited[node] = false
}
