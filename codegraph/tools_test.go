package codegraph

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kartographer/mcp"
)

func toolByName(t *testing.T, ix *Index, name string) mcp.Tool {
	t.Helper()
	for _, tool := range Tools(ix) {
		if tool.Name == name {
			return tool
		}
	}
	require.Failf(t, "missing tool", "%s", name)
	return mcp.Tool{}
}

func TestToolNamesMatchGraphify(t *testing.T) {
	ix := fixture(t)
	var got []string
	for _, tool := range Tools(ix) {
		got = append(got, tool.Name)
	}
	assert.ElementsMatch(t, []string{
		"graphify_find", "graphify_callers", "graphify_callees", "graphify_impact",
		"graphify_tests_for", "graph_stats", "list_repositories",
	}, got)
}

func TestToolHandlers(t *testing.T) {
	ix := fixture(t)

	out, err := toolByName(t, ix, "graphify_callers").Handle(json.RawMessage(`{"symbol":"leaf"}`))
	require.NoError(t, err)
	assert.Contains(t, out, `"name":"example.com/fx/pkg.helper"`)

	out, err = toolByName(t, ix, "graphify_tests_for").Handle(json.RawMessage(`{"target":"leaf"}`))
	require.NoError(t, err)
	assert.Contains(t, out, "TestEntry")

	out, err = toolByName(t, ix, "graphify_find").Handle(json.RawMessage(`{"term":"Entry","limit":1}`))
	require.NoError(t, err)
	assert.Contains(t, out, "Entry")

	out, err = toolByName(t, ix, "graph_stats").Handle(json.RawMessage(`{}`))
	require.NoError(t, err)
	assert.Contains(t, out, `"symbols":5`)

	out, err = toolByName(t, ix, "list_repositories").Handle(nil)
	require.NoError(t, err)
	assert.Contains(t, out, "example.com/fx")
}

func TestToolArgumentErrors(t *testing.T) {
	ix := fixture(t)
	_, err := toolByName(t, ix, "graphify_find").Handle(json.RawMessage(`{}`))
	assert.ErrorContains(t, err, "term is required")

	_, err = toolByName(t, ix, "graphify_callers").Handle(json.RawMessage(`{"symbol":"nope"}`))
	assert.ErrorContains(t, err, "no symbol matches")

	_, err = toolByName(t, ix, "graphify_callers").Handle(json.RawMessage(`not json`))
	assert.Error(t, err)
}
