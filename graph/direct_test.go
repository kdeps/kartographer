package graph

import (
	"bytes"
	"testing"

	"github.com/charmbracelet/log"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestListDirectDependenciesA(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := log.Default()
	dependencies := map[string][]string{
		"A": {"B", "C"},
		"B": {"C"},
		"C": {},
	}

	dg := NewDependencyGraph(fs, logger, dependencies)

	output := captureOutput(func() {
		dg.ListDirectDependencies("A")
	})

	expectedOutput := "A\nA -> B\nA -> B -> C\n"

	assert.Equal(t, expectedOutput, output)
}

func TestListDirectDependenciesB(t *testing.T) {
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

	output := captureOutput(func() {
		dg.ListDirectDependencies("A")
	})

	expectedOutput := "A\nA -> B\nA -> B -> D\nA -> C\n"

	assert.Equal(t, expectedOutput, output)
}
