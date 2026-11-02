package codegraph

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func fixture(t *testing.T) *Index {
	t.Helper()
	ix, err := Build("testdata/fixture")
	require.NoError(t, err)
	return ix
}

func TestBuildIndexesModuleSymbols(t *testing.T) {
	ix := fixture(t)
	assert.Equal(t, "example.com/fx", ix.Module)
	symbols, edges := ix.Stats()
	assert.Equal(t, 5, symbols)
	assert.Positive(t, edges)
}

func TestFindMatchesSubstring(t *testing.T) {
	ix := fixture(t)
	got := names(ix.Find("leaf", 0))
	assert.Equal(t, []string{"example.com/fx/pkg.leaf"}, got)
	assert.Len(t, ix.Find("leaf", 0), 1)
	assert.Len(t, ix.Find("e", 2), 2, "limit caps the result")
}

func TestCallersAndCallees(t *testing.T) {
	ix := fixture(t)
	callers, err := ix.Callers("leaf", 0)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"example.com/fx/pkg.helper", "example.com/fx/pkg.T.Method"}, names(callers))

	callees, err := ix.Callees("Entry", 0)
	require.NoError(t, err)
	assert.Equal(t, []string{"example.com/fx/pkg.helper"}, names(callees))
}

func TestNeighboursCombineCallersAndCallees(t *testing.T) {
	ix := fixture(t)
	got, err := ix.Neighbours("helper", 0)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"example.com/fx/pkg.Entry", "example.com/fx/pkg.leaf"}, names(got))
}

func TestTestsForSymbolAndFile(t *testing.T) {
	ix := fixture(t)
	tests, err := ix.TestsFor("leaf", 0)
	require.NoError(t, err)
	assert.Equal(t, []string{"example.com/fx/pkg.TestEntry"}, names(tests))

	byFile, err := ix.TestsFor("pkg/a.go", 0)
	require.NoError(t, err)
	assert.Equal(t, []string{"example.com/fx/pkg.TestEntry"}, names(byFile))
}

func TestResolveErrors(t *testing.T) {
	ix := fixture(t)
	_, err := ix.Callers("nope", 0)
	assert.ErrorContains(t, err, "no symbol matches")

	_, err = ix.Callers("pkg.T.Method", 0)
	assert.NoError(t, err, "a suffix resolves when unique")
}

func TestBuildFailsOnMissingDir(t *testing.T) {
	_, err := Build("testdata/does-not-exist")
	assert.Error(t, err)
}
