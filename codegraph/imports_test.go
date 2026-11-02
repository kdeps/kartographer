package codegraph

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestModuleImportsKeepsOnlyInModuleImports(t *testing.T) {
	root := t.TempDir()
	files := map[string]string{
		"go.mod":      "module example.com/m\n\ngo 1.21\n",
		"a/a.go":      "package a\n\nimport (\n\t\"fmt\"\n\t\"example.com/m/b\"\n)\n\nvar _ = fmt.Sprint\nvar _ = b.X\n",
		"b/b.go":      "package b\n\nvar X = 1\n",
		"b/broken.go": "package b\n\nimport (\n\t\"example.com/m/a\"\n",
		"root.go":     "package m\n",
	}
	for name, body := range files {
		require.NoError(t, os.MkdirAll(filepath.Dir(filepath.Join(root, name)), 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(root, name), []byte(body), 0o600))
	}

	got, err := ModuleImports(root, []string{"a/a.go", "b/b.go", "b/broken.go", "root.go"})
	require.NoError(t, err)
	require.Len(t, got, 3, "a file that does not parse is left out")

	assert.Equal(t, "a/a.go", got[0].File)
	assert.Equal(t, "example.com/m/a", got[0].Package)
	assert.Equal(t, []string{"example.com/m/b"}, got[0].Imports, "fmt is outside the module")
	assert.Equal(t, []int{5}, got[0].Lines)

	assert.Empty(t, got[1].Imports)
	assert.Equal(t, "example.com/m", got[2].Package, "a file at the root is in the module's own package")
}

func TestModuleImportsNeedsAModule(t *testing.T) {
	_, err := ModuleImports(t.TempDir(), nil)
	assert.Error(t, err)
}
