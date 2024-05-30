package graph

func (dg *DependencyGraph) InvertDependencies() map[string][]string {
	inverted := make(map[string][]string)
	for node, deps := range dg.NodeDependencies {
		for _, dep := range deps {
			inverted[dep] = append(inverted[dep], node)
		}
	}
	return inverted
}
