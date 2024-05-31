package graph

import (
	"bytes"
	"testing"

	"github.com/charmbracelet/log"
	"github.com/spf13/afero"
)

func TestBuildDependencyStack(t *testing.T) {
	fs := afero.NewMemMapFs()
	var buf bytes.Buffer
	logger := log.NewWithOptions(&buf, log.Options{})
	dependencies := map[string][]string{
		"A": {"B", "C"},
		"B": {"D"},
		"C": {"D"},
		"D": {},
	}
	dg := NewDependencyGraph(fs, logger, dependencies)
	visited := make(map[string]bool)
	stack := dg.BuildDependencyStack("A", visited)
	expected := []string{"D", "B", "C", "A"}
	if !equal(stack, expected) {
		t.Errorf("Expected %v, got %v", expected, stack)
	}
}

func TestBuildDependencyStack_Circular(t *testing.T) {
	fs := afero.NewMemMapFs()
	var buf bytes.Buffer
	logger := log.NewWithOptions(&buf, log.Options{})
	dependencies := map[string][]string{
		"A": {"B", "C"},
		"B": {"D"},
		"C": {"D"},
		"D": {"A"}, // Circular dependency here
	}
	dg := NewDependencyGraph(fs, logger, dependencies)
	visited := make(map[string]bool)
	stack := dg.BuildDependencyStack("A", visited)

	expected := []string{"D", "B", "C", "A"}

	if !equal(stack, expected) {
		t.Errorf("Expected %v, got %v", expected, stack)
	}
}
