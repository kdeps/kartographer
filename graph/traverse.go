package graph

func (dg *DependencyGraph) TraverseDependencyGraph(node string, dependencies map[string][]string, visited map[string]bool) {
	if visited[node] {
		return
	}
	if !dg.prepareTraversal(node, visited) {
		return
	}
	dg.processNodeDependencies(node, dependencies, visited)
	dg.DependencyGraph = dg.DependencyGraph[:len(dg.DependencyGraph)-1]
}

func (dg *DependencyGraph) prepareTraversal(node string, visited map[string]bool) bool {
	visited[node] = true
	dg.DependencyGraph = append(dg.DependencyGraph, node)
	currentPath := dg.ConstructDependencyPath(dg.DependencyGraph, "forward")
	if dg.VisitedPaths[currentPath] {
		dg.DependencyGraph = dg.DependencyGraph[:len(dg.DependencyGraph)-1]
		return false
	}
	dg.VisitedPaths[currentPath] = true
	dg.PrintDependencyPath(currentPath)
	return true
}

func (dg *DependencyGraph) processNodeDependencies(node string, dependencies map[string][]string, visited map[string]bool) {
	if deps, exists := dependencies[node]; exists {
		for _, dep := range deps {
			dg.TraverseDependencyGraph(dep, dependencies, visited)
		}
	}
}
