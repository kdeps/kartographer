package codegraph

import (
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildIndexesCLanguageWithLanguageServer(t *testing.T) {
	if _, err := exec.LookPath("clangd"); err != nil {
		t.Skip("clangd is not installed")
	}
	ix, err := Build("testdata/clang")
	require.NoError(t, err)
	assert.Empty(t, ix.Missing)

	leaf, ok := ix.Symbols["app.c::leaf"]
	require.True(t, ok, "leaf is indexed under file::name")
	assert.Equal(t, "c", leaf.Language)
	assert.Equal(t, 1, leaf.Line)

	callers, err := ix.Callers("leaf", 0)
	require.NoError(t, err)
	assert.Equal(t, []string{"app.c::helper"}, names(callers))

	callees, err := ix.Callees("entry", 0)
	require.NoError(t, err)
	assert.Equal(t, []string{"app.c::helper"}, names(callees))

	tests, err := ix.TestsFor("leaf", 0)
	require.NoError(t, err)
	assert.Equal(t, []string{"test_app.c::test_entry"}, names(tests))
}

func TestMissingLanguageServersAreReported(t *testing.T) {
	t.Setenv("PATH", "")
	ix, err := Build("testdata/clang")
	require.NoError(t, err)
	assert.Empty(t, ix.Symbols)
	require.Len(t, ix.Missing, 1)
	assert.Contains(t, ix.Missing[0], "c: none of [clangd] is installed")
}
