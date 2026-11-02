package codegraph

import (
	"testing"

	"github.com/kdeps/kartographer/lsp"
	"github.com/stretchr/testify/assert"
)

func TestMergeRefsRecordsLanguageServerUses(t *testing.T) {
	ix := &Index{Symbols: map[string]Symbol{"src/a.py::f": {Name: "src/a.py::f", File: "src/a.py", Line: 1, Language: "python"}}}
	mergeRefs(ix, []lsp.Ref{{Target: "src/a.py::f", From: "src/b.py::g", File: "src/b.py", Line: 3}})

	got, err := ix.References("src/a.py::f", 0)
	assert.NoError(t, err)
	assert.Equal(t, []Reference{{Symbol: "src/a.py::f", From: "src/b.py::g", File: "src/b.py", Line: 3}}, got)
}

func TestMergeRefsKeepsGoReferencesAndAddsMore(t *testing.T) {
	ix := &Index{
		Symbols: map[string]Symbol{"src/a.py::f": {Name: "src/a.py::f"}},
		refs:    map[string][]Reference{"src/a.py::f": {{Symbol: "src/a.py::f", From: "x", File: "x.go", Line: 1}}},
	}
	mergeRefs(ix, []lsp.Ref{{Target: "src/a.py::f", From: "src/b.py::g", File: "src/b.py", Line: 3}})
	assert.Len(t, ix.refs["src/a.py::f"], 2)
}
