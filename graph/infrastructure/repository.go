package infrastructure

import "github.com/kdeps/kartographer/graph/domain"

type InMemoryGraphRepository struct {
	dependencies map[string][]string
}

func NewInMemoryGraphRepository(deps map[string][]string) domain.GraphRepository {
	return &InMemoryGraphRepository{
		dependencies: deps,
	}
}

func (r *InMemoryGraphRepository) GetNodeDependencies(nodeID string) ([]string, bool) {
	deps, exists := r.dependencies[nodeID]
	return deps, exists
}

func (r *InMemoryGraphRepository) GetAllDependencies() map[string][]string {
	result := make(map[string][]string)
	for k, v := range r.dependencies {
		result[k] = make([]string, len(v))
		copy(result[k], v)
	}
	return result
}

func (r *InMemoryGraphRepository) GetReverseDependencies() map[string][]string {
	reversed := make(map[string][]string)
	for node, deps := range r.dependencies {
		for _, dep := range deps {
			reversed[dep] = append(reversed[dep], node)
		}
	}
	return reversed
}