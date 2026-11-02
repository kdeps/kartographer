package codegraph

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGoFilesWalksOutsideSkippedFolders(t *testing.T) {
	root := t.TempDir()
	for _, f := range []string{"a.go", "pkg/b.go", "vendor/v.go", "testdata/t.go", ".hidden/h.go", "node_modules/n.go", "notes.txt"} {
		require.NoError(t, os.MkdirAll(filepath.Dir(filepath.Join(root, f)), 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(root, f), []byte("package x\n"), 0o600))
	}
	got, err := GoFiles(t.Context(), root)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{filepath.Join(root, "a.go"), filepath.Join(root, "pkg", "b.go")}, got)
}

func TestSkippedPathLooksAtFolderNamesOnly(t *testing.T) {
	assert.True(t, skippedPath("vendor/x/y.go"))
	assert.True(t, skippedPath("a/testdata/y.go"))
	assert.True(t, skippedPath(".cache/y.go"))
	assert.False(t, skippedPath("a/vendored/y.go"), "a folder that only contains the name is not skipped")
	assert.False(t, skippedPath("y.go"))
}

func TestGoFilesNeedsRules(t *testing.T) {
	saved := configured
	configured = false
	t.Cleanup(func() { configured = saved })
	_, err := GoFiles(t.Context(), t.TempDir())
	assert.ErrorIs(t, err, ErrNotConfigured)
}
