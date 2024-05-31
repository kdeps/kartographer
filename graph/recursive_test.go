package graph

import (
	"bytes"
	"testing"

	"github.com/charmbracelet/log"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestListDependenciesRecursive(t *testing.T) {
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
	output := captureOutput(func() {
		dg.ListDependenciesRecursive("A", []string{}, visited)
	})

	expectedOutput := "A <- B <- D\nA <- C <- D\n"

	assert.Equal(t, expectedOutput, output)
}

func TestListDependenciesRecursive_Circular(t *testing.T) {
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
	output := captureOutput(func() {
		dg.ListDependenciesRecursive("A", []string{}, visited)
	})

	expectedOutput := "" // Expected behaviour as dependency_tree should not display redundant resource

	assert.Equal(t, expectedOutput, output)
}
