package usecase

import "github.com/kdeps/kartographer/graph/domain"

type DependencyServiceImpl struct {
	repo      domain.GraphRepository
	pathSvc   domain.PathService
	traversal *domain.GraphTraversal
}

func NewDependencyService(repo domain.GraphRepository, pathSvc domain.PathService) domain.DependencyService {
	return &DependencyServiceImpl{
		repo:      repo,
		pathSvc:   pathSvc,
		traversal: domain.NewGraphTraversal(),
	}
}

func (s *DependencyServiceImpl) ListDirectDependencies(nodeID string) []string {
	deps, exists := s.repo.GetNodeDependencies(nodeID)
	if !exists {
		return []string{}
	}
	return deps
}

func (s *DependencyServiceImpl) ListRecursiveDependencies(nodeID string) []string {
	result := []string{}
	s.collectRecursiveDeps(nodeID, &result, make(map[string]bool))
	return result
}

func (s *DependencyServiceImpl) collectRecursiveDeps(nodeID string, result *[]string, visited map[string]bool) {
	if visited[nodeID] {
		return
	}
	visited[nodeID] = true
	deps, exists := s.repo.GetNodeDependencies(nodeID)
	if !exists {
		return
	}
	for _, dep := range deps {
		*result = append(*result, dep)
		s.collectRecursiveDeps(dep, result, visited)
	}
}

func (s *DependencyServiceImpl) ListReverseDependencies(nodeID string) []string {
	reversed := s.repo.GetReverseDependencies()
	if deps, exists := reversed[nodeID]; exists {
		return deps
	}
	return []string{}
}

func (s *DependencyServiceImpl) BuildDependencyStack(nodeID string) []string {
	stack := []string{}
	s.buildStack(nodeID, &stack, make(map[string]bool))
	return stack
}

func (s *DependencyServiceImpl) buildStack(nodeID string, stack *[]string, visited map[string]bool) {
	if visited[nodeID] {
		return
	}
	visited[nodeID] = true
	deps, exists := s.repo.GetNodeDependencies(nodeID)
	if exists {
		for _, dep := range deps {
			s.buildStack(dep, stack, visited)
		}
	}
	*stack = append(*stack, nodeID)
}

func (s *DependencyServiceImpl) TraverseGraph(nodeID string) {
	s.traversal = domain.NewGraphTraversal()
	s.traverseNode(nodeID)
}

func (s *DependencyServiceImpl) traverseNode(nodeID string) {
	if s.traversal.VisitedNodes[nodeID] {
		return
	}
	s.traversal.VisitedNodes[nodeID] = true
	s.traversal.CurrentPath = append(s.traversal.CurrentPath, nodeID)
	pathStr := s.pathSvc.ConstructPath(s.traversal.CurrentPath, "forward")
	if s.traversal.VisitedPaths[pathStr] {
		s.traversal.CurrentPath = s.traversal.CurrentPath[:len(s.traversal.CurrentPath)-1]
		return
	}
	s.traversal.VisitedPaths[pathStr] = true
	s.pathSvc.PrintPath(s.traversal.CurrentPath, "forward")
	deps, exists := s.repo.GetNodeDependencies(nodeID)
	if exists {
		for _, dep := range deps {
			s.traverseNode(dep)
		}
	}
	s.traversal.CurrentPath = s.traversal.CurrentPath[:len(s.traversal.CurrentPath)-1]
}