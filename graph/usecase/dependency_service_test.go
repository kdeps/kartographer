package usecase

import (
	"testing"
	"github.com/kdeps/kartographer/graph/domain"
	"github.com/kdeps/kartographer/graph/infrastructure"
)

func TestNewDependencyService(t *testing.T) {
	deps := map[string][]string{"A": {"B"}}
	repo := infrastructure.NewInMemoryGraphRepository(deps)
	formatter := infrastructure.NewArrowPathFormatter()
	writer := &mockOutputWriter{}
	pathSvc := NewPathService(formatter, writer)
	
	service := NewDependencyService(repo, pathSvc)
	if service == nil {
		t.Error("Expected service to be created")
	}
}

func TestListDirectDependencies(t *testing.T) {
	deps := map[string][]string{
		"A": {"B", "C"},
		"B": {"D"},
	}
	service := createTestService(deps)
	
	result := service.ListDirectDependencies("A")
	if len(result) != 2 {
		t.Errorf("Expected 2 dependencies, got %d", len(result))
	}
	
	result = service.ListDirectDependencies("NONEXISTENT")
	if len(result) != 0 {
		t.Errorf("Expected 0 dependencies for non-existent node, got %d", len(result))
	}
}

type mockOutputWriter struct {
	lines []string
}

func (m *mockOutputWriter) WriteLine(content string) {
	m.lines = append(m.lines, content)
}

func createTestService(deps map[string][]string) domain.DependencyService {
	repo := infrastructure.NewInMemoryGraphRepository(deps)
	formatter := infrastructure.NewArrowPathFormatter()
	writer := &mockOutputWriter{}
	pathSvc := NewPathService(formatter, writer)
	return NewDependencyService(repo, pathSvc)
}

func TestListRecursiveDependencies(t *testing.T) {
	deps := map[string][]string{
		"A": {"B", "C"},
		"B": {"D"},
		"C": {"D"},
		"D": {},
	}
	service := createTestService(deps)
	
	result := service.ListRecursiveDependencies("A")
	if len(result) < 2 {
		t.Errorf("Expected at least 2 recursive dependencies, got %d", len(result))
	}
}

func TestListReverseDependencies(t *testing.T) {
	deps := map[string][]string{
		"A": {"B"},
		"C": {"B"},
	}
	service := createTestService(deps)
	
	result := service.ListReverseDependencies("B")
	if len(result) != 2 {
		t.Errorf("Expected 2 reverse dependencies, got %d", len(result))
	}
}

func TestBuildDependencyStack(t *testing.T) {
	deps := map[string][]string{
		"A": {"B"},
		"B": {"C"},
		"C": {},
	}
	service := createTestService(deps)
	
	result := service.BuildDependencyStack("A")
	if len(result) != 3 {
		t.Errorf("Expected 3 items in stack, got %d", len(result))
	}
	if result[len(result)-1] != "A" {
		t.Errorf("Expected A to be last in stack, got %s", result[len(result)-1])
	}
}

func TestTraverseGraph(t *testing.T) {
	deps := map[string][]string{
		"A": {"B"},
		"B": {},
	}
	service := createTestService(deps)
	
	service.TraverseGraph("A")
}

func TestListRecursiveDependencies_NonExistent(t *testing.T) {
	deps := map[string][]string{"A": {"B"}}
	service := createTestService(deps)
	
	result := service.ListRecursiveDependencies("NONEXISTENT")
	if len(result) != 0 {
		t.Errorf("Expected 0 dependencies for non-existent node, got %d", len(result))
	}
}

func TestListReverseDependencies_NonExistent(t *testing.T) {
	deps := map[string][]string{"A": {"B"}}
	service := createTestService(deps)
	
	result := service.ListReverseDependencies("NONEXISTENT")
	if len(result) != 0 {
		t.Errorf("Expected 0 reverse dependencies for non-existent node, got %d", len(result))
	}
}

func TestBuildDependencyStack_Visited(t *testing.T) {
	deps := map[string][]string{"A": {"A"}}
	service := createTestService(deps)
	
	result := service.BuildDependencyStack("A")
	if len(result) != 1 {
		t.Errorf("Expected 1 item in stack for self-reference, got %d", len(result))
	}
}

func TestTraverseGraph_Visited(t *testing.T) {
	deps := map[string][]string{"A": {"A"}}
	service := createTestService(deps)
	
	service.TraverseGraph("A")
}

func TestTraverseGraph_VisitedPath(t *testing.T) {
	deps := map[string][]string{
		"A": {"B", "C"},
		"B": {"D"},
		"C": {"D"},
	}
	service := createTestService(deps).(*DependencyServiceImpl)
	service.traversal.VisitedPaths["A -> B"] = true
	
	service.TraverseGraph("A")
}