package graph

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConstructDependencyPath(t *testing.T) {
	dg := &DependencyGraph{}
	path := []string{"A", "B", "C"}
	result := dg.ConstructDependencyPath(path, "forward")
	expected := "A -> B -> C"
	if result != expected {
		t.Errorf("Expected %v, got %v", expected, result)
	}
}

func TestPrintDependencyPath(t *testing.T) {
	dg := &DependencyGraph{}
	path := []string{"A", "B", "C"}
	output := captureOutput(func() {
		dg.PrintDependencyPath(dg.ConstructDependencyPath(path, "forward"))
	})

	expectedOutput := "A -> B -> C\n"

	assert.Equal(t, expectedOutput, output)
}
