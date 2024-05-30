package graph

import "fmt"

func (dg *DependencyGraph) ListDependencyTreeTopDown(node string) {
	visited := make(map[string]bool)
	stack := dg.BuildDependencyStack(node, visited)
	for _, node := range stack {
		fmt.Println(node)
	}
}
