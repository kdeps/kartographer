package graph

import (
	"testing"

	"github.com/charmbracelet/log"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestListReverseDependencies(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := log.Default()
	dependencies := map[string][]string{
		"A": {"B", "C"},
		"B": {"C"},
		"C": {},
	}

	dg := NewDependencyGraph(fs, logger, dependencies)

	output := captureOutput(func() {
		dg.ListReverseDependencies("B")
	})

	expectedOutput := "B\nB -> A\n"

	assert.Equal(t, expectedOutput, output)
}

func TestListReverseDependencies_Circular(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := log.Default()
	dependencies := map[string][]string{
		"A": {"B", "C"},
		"B": {"C"},
		"C": {"A"}, // Circular dependency here
	}

	dg := NewDependencyGraph(fs, logger, dependencies)

	output := captureOutput(func() {
		dg.ListReverseDependencies("B")
	})

	expectedOutput := "B\nB -> A\n"

	assert.Contains(t, output, expectedOutput)
}
