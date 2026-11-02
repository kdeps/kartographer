package codegraph

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kartographer/notes"
)

func tier3Fixture(t *testing.T) *Index {
	t.Helper()
	ix := fixture(t)
	ix.Notes = notes.Open(filepath.Join(t.TempDir(), "notes.json"))
	return ix
}

func TestMemoryToolsRoundTrip(t *testing.T) {
	ix := tier3Fixture(t)

	out, err := toolByName(t, ix, "remember").Handle(json.RawMessage(`{"text":"leaf must stay pure","tags":["gotcha"]}`))
	require.NoError(t, err)
	assert.Contains(t, out, `"saved":true`)

	out, err = toolByName(t, ix, "recall").Handle(json.RawMessage(`{"query":"leaf pure"}`))
	require.NoError(t, err)
	assert.Contains(t, out, "leaf must stay pure")

	out, err = toolByName(t, ix, "memories_about").Handle(json.RawMessage(`{"target":"leaf"}`))
	require.NoError(t, err)
	assert.Contains(t, out, "leaf must stay pure")

	_, err = toolByName(t, ix, "remember").Handle(json.RawMessage(`{}`))
	assert.ErrorContains(t, err, "text is required")
	_, err = toolByName(t, ix, "recall").Handle(json.RawMessage(`{}`))
	assert.ErrorContains(t, err, "query is required")
	_, err = toolByName(t, ix, "memories_about").Handle(json.RawMessage(`{}`))
	assert.ErrorContains(t, err, "target is required")
}

func TestWorkspaceTools(t *testing.T) {
	ix := tier3Fixture(t)
	out, err := toolByName(t, ix, "list_workspaces").Handle(nil)
	require.NoError(t, err)
	assert.Contains(t, out, `"active":"example.com/fx"`)

	out, err = toolByName(t, ix, "set_workspace").Handle(json.RawMessage(`{"handle":"example.com/fx"}`))
	require.NoError(t, err)
	assert.Contains(t, out, "example.com/fx")

	_, err = toolByName(t, ix, "set_workspace").Handle(json.RawMessage(`{"handle":"other"}`))
	assert.ErrorContains(t, err, "unknown workspace")
}

func TestImpactRiskRanksFilesAndListsTests(t *testing.T) {
	ix := tier3Fixture(t)
	files, tests, err := ix.ImpactRisk("leaf", 0)
	require.NoError(t, err)
	require.NotEmpty(t, files)
	assert.Equal(t, "pkg/a.go", files[0]["file"])
	assert.Equal(t, []string{fxTest}, names(tests))

	out, err := toolByName(t, ix, "impact_and_risk").Handle(json.RawMessage(`{"target":"leaf"}`))
	require.NoError(t, err)
	assert.Contains(t, out, `"tests"`)
}

func TestFetchReturnsSymbolSource(t *testing.T) {
	ix := tier3Fixture(t)
	out, err := toolByName(t, ix, "fetch").Handle(json.RawMessage(`{"id":"leaf"}`))
	require.NoError(t, err)
	assert.Contains(t, out, "func leaf")
	assert.Contains(t, out, `"file":"pkg/a.go"`)
}
