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
		"graphify_tests_for", "graphify_rank_files", "query_graph", "graphify_file_neighbors",
		"graph_stats", "list_repositories",
		"graphify_node", "graphify_trace", "shortest_path", "graphify_expand",
		"graphify_find_seeds", "search", "graphify_render_subgraph",
		"remember", "recall", "memories_about", "list_workspaces", "set_workspace", "impact_and_risk", "fetch",
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

func TestQueryAndRankHandlers(t *testing.T) {
	ix := fixture(t)

	out, err := toolByName(t, ix, "query_graph").Handle(json.RawMessage(`{"question":"where is leaf defined","k":1}`))
	require.NoError(t, err)
	assert.Contains(t, out, `"name":"example.com/fx/pkg.leaf"`)
	assert.Contains(t, out, `"body":"func leaf`)

	out, err = toolByName(t, ix, "query_graph").Handle(json.RawMessage(`{"question":"leaf","skeleton":true}`))
	require.NoError(t, err)
	assert.NotContains(t, out, `"body"`)

	out, err = toolByName(t, ix, "graphify_rank_files").Handle(json.RawMessage(`{"question":"leaf"}`))
	require.NoError(t, err)
	assert.Contains(t, out, `"file":"pkg/a.go"`)

	_, err = toolByName(t, ix, "query_graph").Handle(json.RawMessage(`{}`))
	assert.ErrorContains(t, err, "question is required")
	_, err = toolByName(t, ix, "graphify_rank_files").Handle(json.RawMessage(`{}`))
	assert.ErrorContains(t, err, "question is required")
	_, err = toolByName(t, ix, "graphify_file_neighbors").Handle(json.RawMessage(`{}`))
	assert.ErrorContains(t, err, "file is required")

	out, err = toolByName(t, ix, "graphify_file_neighbors").Handle(json.RawMessage(`{"file":"pkg/a.go","symbols_only":true}`))
	require.NoError(t, err)
	assert.Contains(t, out, "example.com/fx/pkg.leaf")
}

func TestWordsDropStopwordsAndPunctuation(t *testing.T) {
	assert.Equal(t, []string{"leaf", "helper_fn"}, Words("Where is the leaf, helper_fn?"))
}

func TestQueryRanksNameMatchesAndCapsCount(t *testing.T) {
	ix := fixture(t)
	hits := ix.Query("entry", 1, 5000, true)
	require.Len(t, hits, 1)
	assert.Equal(t, "example.com/fx/pkg.Entry", hits[0].Symbol.Name)

	assert.Empty(t, ix.Query("zzz-no-match", 5, 5000, false))
}

func TestQueryBodyRespectsBudget(t *testing.T) {
	ix := fixture(t)
	hits := ix.Query("leaf", 5, 10, false)
	require.NotEmpty(t, hits)
	assert.LessOrEqual(t, len(hits[0].Body), 10)
}

func TestRankFilesAggregatesSymbols(t *testing.T) {
	ix := fixture(t)
	hits := ix.RankFiles("pkg a leaf", 0)
	require.NotEmpty(t, hits)
	assert.Equal(t, "pkg/a.go", hits[0].Symbol.File)
	assert.Len(t, ix.RankFiles("leaf", 1), 1)
}

func TestFileNeighboursFromCallGraph(t *testing.T) {
	ix := fixture(t)
	deps, users := ix.FileNeighbors("pkg/a_test.go", 0)
	assert.Equal(t, []string{"pkg/a.go"}, deps)
	assert.Empty(t, users)

	deps, users = ix.FileNeighbors("pkg/a.go", 0)
	assert.Empty(t, deps, "everything in a.go calls within the file")
	assert.Equal(t, []string{"pkg/a_test.go"}, users)
}
