package codegraph

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeWorkspace makes a go.work at the root with module a calling module b.
func writeWorkspace(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	files := map[string]string{
		"go.work":  "go 1.21\n\nuse (\n\t./a\n\t./b\n)\n",
		"a/go.mod": "module example.com/wa\n\ngo 1.21\n",
		"a/a.go":   "package wa\n\nimport \"example.com/wb\"\n\nfunc Use() int { return wb.Helper() }\n",
		"b/go.mod": "module example.com/wb\n\ngo 1.21\n",
		"b/b.go":   "package wb\n\nfunc Helper() int { return 1 }\n",
	}
	for name, body := range files {
		path := filepath.Join(root, name)
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
		require.NoError(t, os.WriteFile(path, []byte(body), 0o600))
	}
	return root
}

func TestGoWorkLinksCallsAcrossModules(t *testing.T) {
	ix, err := Build(writeWorkspace(t))
	require.NoError(t, err)

	assert.Equal(t, "example.com/wa, example.com/wb", ix.Module)
	use, helper := "example.com/wa.Use", "example.com/wb.Helper"
	require.Contains(t, ix.Symbols, use)
	require.Contains(t, ix.Symbols, helper)
	assert.Equal(t, "a/a.go", ix.Symbols[use].File)
	assert.Equal(t, []string{helper}, ix.deps.NodeDependencies[use], "a call into the other module is an edge")
}

func TestGoPackagesWithoutWorkspaceUsesTheModule(t *testing.T) {
	patterns, mods, err := goPackages(filepath.Join("testdata", "fixture"))
	require.NoError(t, err)
	assert.Equal(t, []string{"./..."}, patterns)
	assert.Equal(t, modules{"example.com/fx"}, mods)
}

func TestGoWorkWithNoModulesIsAnError(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "go.work"), []byte("go 1.21\n"), 0o600))
	_, _, err := goPackages(root)
	assert.ErrorIs(t, err, errNoWorkModules)
}

func TestModulesContainsOnlyItsPackages(t *testing.T) {
	ms := modules{"example.com/wa", "example.com/wb"}
	assert.True(t, ms.contains("example.com/wa"))
	assert.True(t, ms.contains("example.com/wb/sub"))
	assert.False(t, ms.contains("example.com/wbx"), "a longer name is another module")
	assert.False(t, ms.contains("fmt"))
}
