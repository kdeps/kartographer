package infrastructure

import "testing"

func TestNewInMemoryGraphRepository(t *testing.T) {
	deps := map[string][]string{
		"A": {"B", "C"},
		"B": {"D"},
		"C": {"D"},
		"D": {},
	}
	
	repo := NewInMemoryGraphRepository(deps)
	if repo == nil {
		t.Error("Expected repository to be created")
	}
}

func TestGetNodeDependencies(t *testing.T) {
	deps := map[string][]string{
		"A": {"B", "C"},
		"B": {"D"},
		"D": {},
	}
	
	repo := NewInMemoryGraphRepository(deps).(*InMemoryGraphRepository)
	
	nodeDeps, exists := repo.GetNodeDependencies("A")
	if !exists {
		t.Error("Expected node A to exist")
	}
	if len(nodeDeps) != 2 {
		t.Errorf("Expected 2 dependencies for A, got %d", len(nodeDeps))
	}
	
	_, exists = repo.GetNodeDependencies("NONEXISTENT")
	if exists {
		t.Error("Expected non-existent node to not exist")
	}
}

func TestGetAllDependencies(t *testing.T) {
	deps := map[string][]string{
		"A": {"B", "C"},
		"B": {"D"},
	}
	
	repo := NewInMemoryGraphRepository(deps).(*InMemoryGraphRepository)
	allDeps := repo.GetAllDependencies()
	
	if len(allDeps) != 2 {
		t.Errorf("Expected 2 nodes, got %d", len(allDeps))
	}
	if len(allDeps["A"]) != 2 {
		t.Errorf("Expected 2 dependencies for A, got %d", len(allDeps["A"]))
	}
}

func TestGetReverseDependencies(t *testing.T) {
	deps := map[string][]string{
		"A": {"B", "C"},
		"B": {"D"},
		"C": {"D"},
	}
	
	repo := NewInMemoryGraphRepository(deps).(*InMemoryGraphRepository)
	reversed := repo.GetReverseDependencies()
	
	if len(reversed["D"]) != 2 {
		t.Errorf("Expected 2 reverse dependencies for D, got %d", len(reversed["D"]))
	}
	if len(reversed["B"]) != 1 {
		t.Errorf("Expected 1 reverse dependency for B, got %d", len(reversed["B"]))
	}
}