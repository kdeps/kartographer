package graph

func (dg *DependencyGraph) ListReverseDependencies(node string) {
	inverted := dg.InvertDependencies()
	dg.DependencyGraph = []string{}
	dg.VisitedPaths = make(map[string]bool)
	visited := make(map[string]bool)
	dg.TraverseDependencyGraph(node, inverted, visited)
}
