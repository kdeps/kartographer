package graph

import (
	"testing"

	"github.com/charmbracelet/log"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

// TestNewDependencyGraph tests the initialization of a new DependencyGraph
func TestNewDependencyGraph(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := log.Default()
	dependencies := map[string][]string{
		"A": {"B", "C"},
		"B": {"C"},
		"C": {},
	}

	dg := NewDependencyGraph(fs, logger, dependencies)

	assert.NotNil(t, dg)
	assert.Equal(t, dependencies, dg.NodeDependencies)
	assert.Empty(t, dg.DependencyGraph)
	assert.Empty(t, dg.VisitedPaths)
}
