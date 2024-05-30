package graph

import (
	"bytes"
	"testing"

	"github.com/charmbracelet/log"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestTraverseDependencyGraph(t *testing.T) {
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
		dg.TraverseDependencyGraph("A", dependencies, visited)
	})

	expectedOutput := "A\nA -> B\nA -> B -> D\nA -> C\n"

	assert.Equal(t, expectedOutput, output)
}
