package graph

import (
	"fmt"
	"strings"
)

func (dg *DependencyGraph) ConstructDependencyPath(path []string, dir string) string {
	var graph string
	switch dir {
	case "forward":
		graph = strings.Join(path, " -> ")
	case "reverse":
		graph = strings.Join(path, " <- ")
	default:
		graph = strings.Join(path, " -> ")
	}

	return graph
}

func (dg *DependencyGraph) PrintDependencyPath(path string) {
	fmt.Println(path)
}
