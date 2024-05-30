package graph

import (
	"bytes"
	"testing"

	"github.com/charmbracelet/log"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestListDependencyTreeTopDown(t *testing.T) {
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
		dg.ListDependencyTreeTopDown("A")
	})

	expectedOutput := "D\nB\nC\nA\n"

	assert.Equal(t, expectedOutput, output)
}
