package graph

func (dg *DependencyGraph) ListDirectDependencies(node string) {
	dg.DependencyGraph = []string{}
	dg.VisitedPaths = make(map[string]bool)
	visited := make(map[string]bool)
	dg.TraverseDependencyGraph(node, dg.NodeDependencies, visited)
}
