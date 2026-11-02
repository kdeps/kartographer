package codegraph

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func refsFixture(t *testing.T) *Index {
	t.Helper()
	ix, err := Build("testdata/refs")
	require.NoError(t, err)
	return ix
}

func TestReferencesFindValueUses(t *testing.T) {
	ix := refsFixture(t)
	refs, err := ix.References("Exported", 0)
	require.NoError(t, err)
	var from []string
	for _, r := range refs {
		from = append(from, r.From)
	}
	assert.Contains(t, from, "example.com/refs/pkg.Run", "passed as a value in Run")
	assert.Contains(t, from, "example.com/refs/pkg.init", "stored in the Table variable")
}

func TestReferencesExcludeDirectCalls(t *testing.T) {
	ix := refsFixture(t)
	refs, err := ix.References("apply", 0)
	require.NoError(t, err)
	assert.Empty(t, refs, "apply is only called, never used as a value")
}

func TestReferencesUnknownSymbol(t *testing.T) {
	ix := refsFixture(t)
	_, err := ix.References("nope", 0)
	assert.ErrorContains(t, err, "no symbol matches")
}

func TestImportsExportsOfGoFile(t *testing.T) {
	ix := refsFixture(t)
	imports, exports, err := ix.ImportsExports("pkg/refs.go")
	require.NoError(t, err)
	assert.Empty(t, imports)
	assert.Equal(t, []string{"Exported", "Run", "Table", "Type"}, exports, "unexported names and methods are left out")

	_, _, err = ix.ImportsExports("README.md")
	assert.ErrorIs(t, err, errNotGo)
}

func TestReferenceToolHandlers(t *testing.T) {
	ix := refsFixture(t)
	var refTools = RefTools(ix)
	require.Len(t, refTools, 2)

	out, err := refTools[0].Handle(json.RawMessage(`{"symbol":"Exported"}`))
	require.NoError(t, err)
	assert.Contains(t, out, `"from":"example.com/refs/pkg.Run"`)

	out, err = refTools[1].Handle(json.RawMessage(`{"file":"pkg/refs.go"}`))
	require.NoError(t, err)
	assert.Contains(t, out, `"exports"`)

	_, err = refTools[1].Handle(json.RawMessage(`{}`))
	assert.ErrorContains(t, err, "file is required")
}
