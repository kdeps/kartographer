package codegraph

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// copyFixture copies the test module into a temporary directory the test may edit.
func copyFixture(t *testing.T) string {
	t.Helper()
	dst := t.TempDir()
	require.NoError(t, filepath.WalkDir("testdata/fixture", func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, _ := filepath.Rel("testdata/fixture", p)
		raw, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		out := filepath.Join(dst, rel)
		if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
			return err
		}
		return os.WriteFile(out, raw, 0o644)
	}))
	return dst
}

func TestBuildCachedReusesUnchangedModule(t *testing.T) {
	src := copyFixture(t)
	cache := t.TempDir()

	first, err := BuildCached(src, cache)
	require.NoError(t, err)
	entries, err := os.ReadDir(cache)
	require.NoError(t, err)
	require.Len(t, entries, 1, "one snapshot is saved")

	second, err := BuildCached(src, cache)
	require.NoError(t, err)
	assert.Equal(t, first.Module, second.Module)
	assert.Equal(t, len(first.Symbols), len(second.Symbols))
	got, err := second.Callers("leaf", 0)
	require.NoError(t, err)
	assert.Equal(t, []string{"example.com/fx/pkg.T.Method", fxHelper}, names(got))
}

func TestBuildCachedRebuildsAfterEdit(t *testing.T) {
	src := copyFixture(t)
	cache := t.TempDir()
	_, err := BuildCached(src, cache)
	require.NoError(t, err)

	file := filepath.Join(src, "pkg", "a.go")
	raw, err := os.ReadFile(file)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(file, append(raw, []byte("\nfunc added() int { return 2 }\n")...), 0o644))
	later := time.Now().Add(time.Minute)
	require.NoError(t, os.Chtimes(file, later, later))

	ix, err := BuildCached(src, cache)
	require.NoError(t, err)
	assert.Contains(t, ix.Symbols, "example.com/fx/pkg.added")
}

func TestBuildCachedIgnoresCorruptSnapshot(t *testing.T) {
	src := copyFixture(t)
	cache := t.TempDir()
	_, err := BuildCached(src, cache)
	require.NoError(t, err)

	entries, err := os.ReadDir(cache)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(cache, entries[0].Name()), []byte("not json"), 0o600))

	ix, err := BuildCached(src, cache)
	require.NoError(t, err)
	assert.Contains(t, ix.Symbols, fxLeaf)
}

func TestCacheDirIsUnderUserCache(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	dir, err := CacheDir()
	require.NoError(t, err)
	assert.Contains(t, dir, "kartographer")
}
