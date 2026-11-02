package codegraph

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChangedLinesReadsNewSideRanges(t *testing.T) {
	diff := "diff --git a/x.go b/x.go\n+++ b/x.go\n@@ -1,2 +4,3 @@\n@@ -9 +20 @@\n"
	got := ChangedLines(diff)
	assert.Equal(t, [][2]int{{4, 6}, {20, 20}}, got["x.go"], "a count left out is one line")
}

func TestChangedLinesMarksARemovalAtItsLine(t *testing.T) {
	got := ChangedLines("+++ b/x.go\n@@ -5,2 +5,0 @@\n")
	assert.Equal(t, [][2]int{{5, 5}}, got["x.go"])
}

func TestChangedLinesSkipsDeletedFilesAndUnreadableHunks(t *testing.T) {
	got := ChangedLines("+++ /dev/null\n@@ -1,3 +0,0 @@\n+++ b/y.go\n@@ -1 +x @@\n")
	assert.Empty(t, got)
}

func TestNewSideReadsTheHeader(t *testing.T) {
	start, count, ok := newSide("@@ -3,2 +5,4 @@ func f()")
	assert.True(t, ok)
	assert.Equal(t, 5, start)
	assert.Equal(t, 4, count)
	_, _, ok = newSide("@@ -1 @@")
	assert.False(t, ok)
}

func TestChangedFindsTheSymbolsAHunkTouches(t *testing.T) {
	ix := fixture(t)
	// Line 5 of pkg/a.go is helper.
	got := ix.Changed("+++ b/pkg/a.go\n@@ -5 +5 @@\n")
	require.Len(t, got, 1)
	assert.Equal(t, "example.com/fx/pkg.helper", got[0].Name)
}

func TestChangedIgnoresLinesWithNoSymbol(t *testing.T) {
	ix := fixture(t)
	assert.Empty(t, ix.Changed("+++ b/pkg/a.go\n@@ -100 +100 @@\n"))
}

func TestImpactAddsTheCallersOfTheChangedSymbols(t *testing.T) {
	ix := fixture(t)
	got := ix.Impact("+++ b/pkg/a.go\n@@ -5 +5 @@\n", 2, 60)
	names := map[string]bool{}
	for _, s := range got {
		names[s.Name] = true
	}
	assert.True(t, names["example.com/fx/pkg.helper"], "the changed symbol itself")
	assert.True(t, names["example.com/fx/pkg.Entry"], "its caller is affected")
}

func TestChangesToolReturnsImpactAsJSON(t *testing.T) {
	ix := fixture(t)
	var tool func(json.RawMessage) (string, error)
	for _, tl := range ChangesTools(ix) {
		if tl.Name == "impact_of_diff" {
			tool = tl.Handle
		}
	}
	require.NotNil(t, tool)
	out, err := tool(json.RawMessage(`{"diff":"+++ b/pkg/a.go\n@@ -5 +5 @@\n"}`))
	require.NoError(t, err)
	assert.Contains(t, out, `"name":"example.com/fx/pkg.helper"`)

	_, err = tool(json.RawMessage(`{}`))
	assert.ErrorContains(t, err, "diff is required")
}
