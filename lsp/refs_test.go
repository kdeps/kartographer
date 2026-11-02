package lsp

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func sampleFuncs() []Func {
	return []Func{
		{Key: "a.py::outer", File: "a.py", Name: "outer", Line: 1, End: 10},
		{Key: "a.py::outer.inner", File: "a.py", Name: "outer.inner", Line: 3, End: 5},
		{Key: "b.py::use", File: "b.py", Name: "use", Line: 1, End: 4},
	}
}

func loc(uri string, line int) location {
	return location{URI: uri, Range: span{Start: position{Line: line}}}
}

func TestEnclosingPicksTheInnermostBody(t *testing.T) {
	f, ok := enclosing(sampleFuncs(), "a.py", 4)
	assert.True(t, ok)
	assert.Equal(t, "a.py::outer.inner", f.Key, "a line inside a nested body belongs to the nested function")

	f, ok = enclosing(sampleFuncs(), "a.py", 7)
	assert.True(t, ok)
	assert.Equal(t, "a.py::outer", f.Key, "a line after the nested body is still in the outer one")
}

func TestEnclosingIgnoresLinesOutsideEveryBody(t *testing.T) {
	_, ok := enclosing(sampleFuncs(), "a.py", 12)
	assert.False(t, ok, "a line past every body has no enclosing function")

	_, ok = enclosing(sampleFuncs(), "c.py", 2)
	assert.False(t, ok, "a file with no functions has no enclosing function")
}

func TestReferenceSitesKeepsNonCallUsesOnly(t *testing.T) {
	calls := map[string][]string{"b.py::use": {"a.py::outer"}}
	relOf := map[string]string{"file:///r/a.py": "a.py", "file:///r/b.py": "b.py"}
	locs := []location{
		loc("file:///r/b.py", 2), // inside use, and use calls outer: already an edge
		loc("file:///r/a.py", 4), // a value use inside the nested function
		loc("file:///r/x.py", 1), // outside the module's files
	}

	got := referenceSites(sampleFuncs(), calls, "a.py::outer", relOf, locs)
	assert.Equal(t, []Ref{{Target: "a.py::outer", From: "a.py::outer.inner", File: "a.py", Line: 5}}, got)
}

func TestReferenceSitesKeepsAUseThatIsNotACall(t *testing.T) {
	relOf := map[string]string{"file:///r/b.py": "b.py"}
	got := referenceSites(sampleFuncs(), nil, "a.py::outer", relOf, []location{loc("file:///r/b.py", 1)})
	assert.Equal(t, []Ref{{Target: "a.py::outer", From: "b.py::use", File: "b.py", Line: 2}}, got)
}
