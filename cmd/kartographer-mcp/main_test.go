package main

import (
	"testing"

	"github.com/kdeps/kartographer/codegraph"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// configure gives the command the rules it needs; the tests use no real policy.
func configure(t *testing.T) {
	t.Helper()
	require.NoError(t, codegraph.Configure([]byte("ranking:\n  name_weight: 3\n  path_weight: 1\n  degree_weight: 0.1\n  body_lines: 30\n")))
}

func TestLoadBuildsFreshWithNoCache(t *testing.T) {
	configure(t)
	ix, err := load("../../codegraph/testdata/fixture", true)
	require.NoError(t, err)
	assert.Equal(t, "example.com/fx", ix.Module)
	assert.NotEmpty(t, ix.Symbols)
}

func TestLoadUsesSavedIndex(t *testing.T) {
	configure(t)
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	first, err := load("../../codegraph/testdata/fixture", false)
	require.NoError(t, err)
	second, err := load("../../codegraph/testdata/fixture", false)
	require.NoError(t, err)
	assert.Equal(t, len(first.Symbols), len(second.Symbols))
}

func TestLoadFailsOnMissingDirectory(t *testing.T) {
	configure(t)
	_, err := load("../../codegraph/testdata/nope", true)
	assert.Error(t, err)
}
