package knowledge

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLinkIsNotReplacedByAGuess(t *testing.T) {
	b := NewBuilder()
	b.Link("a", "b", "references", "a.md", "3")
	b.Infer("a", "b", "references", 0.99, 0.5, "x.md", "9")
	g := b.Graph()
	assert.Len(t, g.Edges, 1)
	assert.Equal(t, ProvExtracted, g.Edges[0].Provenance)
	assert.Equal(t, "a.md", g.Edges[0].SourceFile)
}

func TestMoreConfidentGuessReplacesLessConfident(t *testing.T) {
	b := NewBuilder()
	b.Infer("a", "b", "relates_to", 0.6, 0.5, "x.md", "1")
	b.Infer("a", "b", "relates_to", 0.8, 0.5, "y.md", "2")
	b.Infer("a", "b", "relates_to", 0.7, 0.5, "z.md", "3")
	g := b.Graph()
	assert.Len(t, g.Edges, 1)
	assert.InDelta(t, 0.8, g.Edges[0].Confidence, 1e-9)
	assert.Equal(t, "y.md", g.Edges[0].SourceFile, "the evidence of the kept guess is kept")
}

func TestGuessBelowThresholdIsAmbiguous(t *testing.T) {
	b := NewBuilder()
	b.Infer("a", "b", "relates_to", 0.3, 0.5, "x.md", "1")
	b.Infer("a", "c", "relates_to", 0.7, 0.5, "x.md", "2")
	g := b.Graph()
	assert.Equal(t, ProvAmbiguous, g.Edges[0].Provenance)
	assert.Equal(t, ProvInferred, g.Edges[1].Provenance)
}

func TestSelfEdgesAreDropped(t *testing.T) {
	b := NewBuilder()
	b.Link("a", "a", "references", "a.md", "1")
	b.Infer("a", "a", "relates_to", 0.9, 0.5, "a.md", "1")
	assert.Empty(t, b.Graph().Edges)
}

func TestAddKeepsTheFirstNodeWithAnID(t *testing.T) {
	b := NewBuilder()
	first := b.Add(&Node{ID: "n", Label: "first"})
	again := b.Add(&Node{ID: "n", Label: "second"})
	assert.Same(t, first, again)
	assert.Equal(t, "first", b.Graph().Nodes[0].Label)
}

func TestGraphIsSorted(t *testing.T) {
	b := NewBuilder()
	b.Add(&Node{ID: "b"})
	b.Add(&Node{ID: "a"})
	b.Link("b", "a", "z", "", "")
	b.Link("a", "c", "y", "", "")
	b.Link("a", "b", "x", "", "")
	g := b.Graph()
	assert.Equal(t, "a", g.Nodes[0].ID)
	assert.Equal(t, "a", g.Edges[0].From)
	assert.Equal(t, "b", g.Edges[0].To)
	assert.Equal(t, "b", g.Edges[2].From)
}

func TestCommunitiesSplitTwoCliquesWithWeights(t *testing.T) {
	b := NewBuilder()
	ids := []string{"a", "b", "c", "d", "e", "f"}
	for _, id := range ids {
		b.Add(&Node{ID: id})
	}
	pairs := [][2]string{{"a", "b"}, {"a", "c"}, {"b", "c"}, {"d", "e"}, {"d", "f"}, {"e", "f"}, {"c", "d"}}
	for _, p := range pairs {
		b.Link(p[0], p[1], "references", "", "")
	}
	g := b.Graph()
	q := Communities(g, func(*Edge) float64 { return 1 })
	assert.Greater(t, q, 0.0)
	assert.Equal(t, 2, g.Communities)
	comm := map[string]int{}
	for _, n := range g.Nodes {
		comm[n.ID] = n.Community
	}
	assert.Equal(t, comm["a"], comm["c"])
	assert.Equal(t, comm["d"], comm["f"])
	assert.NotEqual(t, comm["a"], comm["d"])
}

func TestCommunitiesOfNothingIsZero(t *testing.T) {
	assert.Equal(t, 0.0, Communities(&Graph{}, func(*Edge) float64 { return 1 }))
}
