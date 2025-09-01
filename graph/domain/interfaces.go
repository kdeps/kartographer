package domain

type GraphRepository interface {
	GetNodeDependencies(nodeID string) ([]string, bool)
	GetAllDependencies() map[string][]string
	GetReverseDependencies() map[string][]string
}

type PathFormatter interface {
	FormatPath(path *Path) string
}

type OutputWriter interface {
	WriteLine(content string)
}

type DependencyService interface {
	ListDirectDependencies(nodeID string) []string
	ListRecursiveDependencies(nodeID string) []string
	ListReverseDependencies(nodeID string) []string
	BuildDependencyStack(nodeID string) []string
	TraverseGraph(nodeID string) 
}

type PathService interface {
	ConstructPath(nodes []string, direction string) string
	PrintPath(nodes []string, direction string)
}