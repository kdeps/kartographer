package codegraph

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	fxEntry  = "example.com/fx/pkg.Entry"
	fxHelper = "example.com/fx/pkg.helper"
	fxLeaf   = "example.com/fx/pkg.leaf"
	fxTest   = "example.com/fx/pkg.TestEntry"
)

func TestTraceFindsDirectedPaths(t *testing.T) {
	ix := fixture(t)
	paths, err := ix.Trace("Entry", "leaf", 3)
	require.NoError(t, err)
	require.Len(t, paths, 1)
	assert.Equal(t, []string{fxEntry, fxHelper, fxLeaf}, paths[0])

	none, err := ix.Trace("leaf", "Entry", 3)
	require.NoError(t, err)
	assert.Empty(t, none, "calls only flow one way")
}

func TestShortestPathFollowsEitherDirection(t *testing.T) {
	ix := fixture(t)
	path, err := ix.ShortestPath("leaf", "TestEntry")
	require.NoError(t, err)
	assert.Equal(t, []string{fxLeaf, fxHelper, fxEntry, fxTest}, path)

	_, err = ix.ShortestPath("leaf", "nope")
	assert.ErrorContains(t, err, "no symbol matches")
}

func TestNeighbourhoodAndNode(t *testing.T) {
	ix := fixture(t)
	nodes, edges := ix.Neighbourhood([]string{"leaf"}, 1, 100)
	assert.Contains(t, nodes, fxLeaf)
	assert.Contains(t, nodes, fxHelper)
	assert.Contains(t, nodes, "example.com/fx/pkg.T.Method")
	assert.Contains(t, edges, [2]string{fxHelper, fxLeaf})

	sym, body, neigh, err := ix.Node("Entry", 12)
	require.NoError(t, err)
	assert.Equal(t, fxEntry, sym.Name)
	assert.Contains(t, body, "func Entry")
	assert.Equal(t, []string{fxTest, fxHelper}, names(neigh))
}

func TestExpandReturnsSourceOrEmpty(t *testing.T) {
	ix := fixture(t)
	got := ix.Expand([]string{"leaf", "nope"})
	assert.Contains(t, got["leaf"], "func leaf")
	assert.Equal(t, "", got["nope"])
}

func TestTier2HandlersAndNames(t *testing.T) {
	ix := fixture(t)
	var got []string
	for _, tool := range Tools(ix) {
		got = append(got, tool.Name)
	}
	for _, want := range []string{"node", "trace", "shortest_path", "expand",
		"find_seeds", "search", "render_subgraph"} {
		assert.Contains(t, got, want)
	}

	out, err := toolByName(t, ix, "trace").Handle(json.RawMessage(`{"src":"Entry","tgt":"leaf"}`))
	require.NoError(t, err)
	assert.Contains(t, out, `"paths"`)

	out, err = toolByName(t, ix, "shortest_path").Handle(json.RawMessage(`{"src":"leaf","tgt":"TestEntry"}`))
	require.NoError(t, err)
	assert.Contains(t, out, `"connected":true`)

	out, err = toolByName(t, ix, "node").Handle(json.RawMessage(`{"symbol":"leaf"}`))
	require.NoError(t, err)
	assert.Contains(t, out, "func leaf")

	out, err = toolByName(t, ix, "render_subgraph").Handle(json.RawMessage(`{"seeds":["leaf"]}`))
	require.NoError(t, err)
	assert.Contains(t, out, `"links"`)

	out, err = toolByName(t, ix, "search").Handle(json.RawMessage(`{"query":"Entry"}`))
	require.NoError(t, err)
	assert.Contains(t, out, fxEntry)

	out, err = toolByName(t, ix, "find_seeds").Handle(json.RawMessage(`{"question":"leaf"}`))
	require.NoError(t, err)
	assert.Contains(t, out, fxLeaf)

	_, err = toolByName(t, ix, "expand").Handle(json.RawMessage(`{}`))
	assert.ErrorContains(t, err, "ids is required")
	_, err = toolByName(t, ix, "render_subgraph").Handle(json.RawMessage(`{}`))
	assert.ErrorContains(t, err, "seeds is required")
	_, err = toolByName(t, ix, "search").Handle(json.RawMessage(`{}`))
	assert.ErrorContains(t, err, "query is required")
}
