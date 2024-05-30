package graph

func (dg *DependencyGraph) TraverseDependencyGraph(node string, dependencies map[string][]string, visited map[string]bool) {
	if visited[node] {
		return
	}
	visited[node] = true
	dg.DependencyGraph = append(dg.DependencyGraph, node)

	currentPath := dg.ConstructDependencyPath(dg.DependencyGraph, "forward")
	if dg.VisitedPaths[currentPath] {
		dg.DependencyGraph = dg.DependencyGraph[:len(dg.DependencyGraph)-1]
		return
	}

	dg.VisitedPaths[currentPath] = true
	dg.PrintDependencyPath(currentPath)

	if deps, exists := dependencies[node]; exists {
		for _, dep := range deps {
			dg.TraverseDependencyGraph(dep, dependencies, visited)
		}
	}
	dg.DependencyGraph = dg.DependencyGraph[:len(dg.DependencyGraph)-1]
}
