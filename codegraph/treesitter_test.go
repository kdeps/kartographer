package codegraph

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIndexTreeSitterLinksCallsByNameAsInferred(t *testing.T) {
	dir := t.TempDir()
	src := "def helper():\n    return 1\n\ndef main():\n    helper()\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "a.py"), []byte(src), 0o600))

	idx := &Index{Symbols: map[string]Symbol{}}
	deps := map[string][]string{}
	require.NoError(t, indexTreeSitter(dir, "python", []string{"a.py"}, idx, deps))

	helper, main := "a.py::helper", "a.py::main"
	require.Contains(t, idx.Symbols, helper)
	require.Contains(t, idx.Symbols, main)
	assert.Equal(t, "python", idx.Symbols[main].Language)
	assert.Equal(t, 4, idx.Symbols[main].Line)
	assert.Equal(t, []string{helper}, deps[main])
	assert.Equal(t, "INFERRED", (&Index{inferred: idx.inferred}).Confidence(main, helper))
}

func TestIndexTreeSitterSkipsSelfCalls(t *testing.T) {
	dir := t.TempDir()
	src := "def again(n):\n    return again(n - 1)\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "r.py"), []byte(src), 0o600))

	idx := &Index{Symbols: map[string]Symbol{}}
	deps := map[string][]string{}
	require.NoError(t, indexTreeSitter(dir, "python", []string{"r.py"}, idx, deps))
	assert.Empty(t, deps["r.py::again"], "a recursive call is not an edge, as with the language servers")
}

func TestIndexTreeSitterReportsAMissingFile(t *testing.T) {
	idx := &Index{Symbols: map[string]Symbol{}}
	err := indexTreeSitter(t.TempDir(), "python", []string{"gone.py"}, idx, map[string][]string{})
	assert.Error(t, err)
}
