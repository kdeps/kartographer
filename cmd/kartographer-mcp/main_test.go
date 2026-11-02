package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadBuildsFreshWithNoCache(t *testing.T) {
	ix, err := load("../../codegraph/testdata/fixture", true)
	require.NoError(t, err)
	assert.Equal(t, "example.com/fx", ix.Module)
	assert.NotEmpty(t, ix.Symbols)
}

func TestLoadUsesSavedIndex(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	first, err := load("../../codegraph/testdata/fixture", false)
	require.NoError(t, err)
	second, err := load("../../codegraph/testdata/fixture", false)
	require.NoError(t, err)
	assert.Equal(t, len(first.Symbols), len(second.Symbols))
}

func TestLoadFailsOnMissingDirectory(t *testing.T) {
	_, err := load("../../codegraph/testdata/nope", true)
	assert.Error(t, err)
}
