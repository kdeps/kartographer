package domain

type Node struct {
	ID           string
	Dependencies []string
}

type Path struct {
	Nodes     []string
	Direction string
}

type GraphTraversal struct {
	VisitedNodes map[string]bool
	VisitedPaths map[string]bool
	CurrentPath  []string
}

func NewNode(id string, dependencies []string) *Node {
	return &Node{
		ID:           id,
		Dependencies: dependencies,
	}
}

func NewPath(nodes []string, direction string) *Path {
	return &Path{
		Nodes:     nodes,
		Direction: direction,
	}
}

func NewGraphTraversal() *GraphTraversal {
	return &GraphTraversal{
		VisitedNodes: make(map[string]bool),
		VisitedPaths: make(map[string]bool),
		CurrentPath:  make([]string, 0),
	}
}