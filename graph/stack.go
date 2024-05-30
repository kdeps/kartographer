package graph

func (dg *DependencyGraph) BuildDependencyStack(node string, visited map[string]bool) []string {
	if visited[node] {
		return nil
	}
	visited[node] = true

	var stack []string
	deps, exists := dg.NodeDependencies[node]
	if exists {
		for _, dep := range deps {
			substack := dg.BuildDependencyStack(dep, visited)
			stack = append(stack, substack...)
		}
	}
	stack = append(stack, node)
	return stack
}
