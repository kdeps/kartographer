package domain

import "testing"

func TestNewNode(t *testing.T) {
	deps := []string{"dep1", "dep2"}
	node := NewNode("test-id", deps)
	
	if node.ID != "test-id" {
		t.Errorf("Expected ID to be 'test-id', got %s", node.ID)
	}
	if len(node.Dependencies) != 2 {
		t.Errorf("Expected 2 dependencies, got %d", len(node.Dependencies))
	}
	if node.Dependencies[0] != "dep1" || node.Dependencies[1] != "dep2" {
		t.Errorf("Dependencies not set correctly")
	}
}

func TestNewPath(t *testing.T) {
	nodes := []string{"A", "B", "C"}
	path := NewPath(nodes, "forward")
	
	if len(path.Nodes) != 3 {
		t.Errorf("Expected 3 nodes, got %d", len(path.Nodes))
	}
	if path.Direction != "forward" {
		t.Errorf("Expected direction 'forward', got %s", path.Direction)
	}
}

func TestNewGraphTraversal(t *testing.T) {
	traversal := NewGraphTraversal()
	
	if traversal.VisitedNodes == nil {
		t.Error("Expected VisitedNodes to be initialized")
	}
	if traversal.VisitedPaths == nil {
		t.Error("Expected VisitedPaths to be initialized")
	}
	if traversal.CurrentPath == nil {
		t.Error("Expected CurrentPath to be initialized")
	}
	if len(traversal.CurrentPath) != 0 {
		t.Errorf("Expected empty CurrentPath, got length %d", len(traversal.CurrentPath))
	}
}