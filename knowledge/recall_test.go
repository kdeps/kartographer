package knowledge

import (
	"testing"

	"github.com/kdeps/kartographer/text"
)

var testTuning = Tuning{
	MaxHops: 2, DefaultBudget: 1000, MaxSeeds: 5, MaxHits: 8, BytesPerToken: 4,
	HopDecay: 0.5, LabelMatchBoost: 2, FullLabelBoost: 3, TagEdgeWeight: 0.3, AmbiguousWeight: 0.3,
}

var testWords = text.Options{MinWordLength: 3, MinStemLength: 3, Stopwords: map[string]bool{"the": true}}

func testIndex(g *Graph) *Index {
	return NewIndex(g, Options{Words: testWords, Tuning: testTuning})
}

func TestEdgeWeight(t *testing.T) {
	cases := []struct {
		name string
		e    Edge
		want float64
	}{
		{"extracted", Edge{Provenance: ProvExtracted}, 1},
		{"inferred uses confidence", Edge{Provenance: ProvInferred, Confidence: 0.4}, 0.4},
		{"ambiguous scales by weight", Edge{Provenance: ProvAmbiguous, Confidence: 0.5}, 0.15},
		{"tag edge scales by tag weight", Edge{Provenance: ProvExtracted, Relation: RelTagged}, 0.3},
	}
	for _, c := range cases {
		if got := EdgeWeight(&c.e, testTuning); got != c.want {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}

func TestRecall_WalksToLinkedNote(t *testing.T) {
	g := &Graph{
		Nodes: []*Node{
			{ID: "note:billing.md", Label: "billing", Note: "billing.md", Summary: "monthly invoices"},
			{ID: "note:rule.md", Label: "rule", Note: "rule.md", Summary: "charge on the first"},
		},
		Edges: []*Edge{{From: "note:billing.md", To: "note:rule.md", Relation: "references",
			Provenance: ProvExtracted, Confidence: 1}},
	}
	r := testIndex(g).Recall("billing invoices", 0, 0)
	if len(r.Hits) != 2 {
		t.Fatalf("want 2 hits, got %d: %+v", len(r.Hits), r.Hits)
	}
	if r.Hits[0].Note != "billing.md" || r.Hits[1].Note != "rule.md" || r.Hits[1].Hops != 1 {
		t.Fatalf("unexpected order or hops: %+v", r.Hits)
	}
	if got := r.Hits[1].Chain(); got != "billing.md --references--> rule.md" {
		t.Errorf("chain = %q", got)
	}
}

func TestRecall_ExcludesAndBudget(t *testing.T) {
	g := &Graph{Nodes: []*Node{{ID: "note:a.md", Label: "billing", Note: "a.md"}}}
	ix := testIndex(g)
	if r := ix.Recall("billing", 1, 0, "a.md"); len(r.Hits) != 0 {
		t.Errorf("excluded note recalled: %+v", r.Hits)
	}
	if r := ix.Recall("billing", 1, 1); len(r.Hits) != 0 {
		t.Errorf("hit over budget kept: %+v", r.Hits)
	}
}

func TestRecall_NodeWeightScalesScore(t *testing.T) {
	g := &Graph{Nodes: []*Node{{ID: "note:s.md", Label: "billing", Note: "s.md", Kind: "session"}}}
	weighted := NewIndex(g, Options{Words: testWords, Tuning: testTuning,
		NodeWeight: func(n *Node) float64 { return 0.5 }})
	plain := testIndex(g)
	w := weighted.Recall("billing", 0, 0).Hits[0].Score
	p := plain.Recall("billing", 0, 0).Hits[0].Score
	if w != p*0.5 {
		t.Errorf("weighted %v, plain %v", w, p)
	}
}

func TestShortID(t *testing.T) {
	for in, want := range map[string]string{
		"note:a.md": "a.md", "tag:billing": "billing", "code:pkg/x": "pkg/x",
		"plain/path:12.md": "plain/path:12.md", "": "",
	} {
		if got := shortID(in); got != want {
			t.Errorf("shortID(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestNeighbours_ExpandsTagsAndSkipsSelf(t *testing.T) {
	g := &Graph{
		Nodes: []*Node{
			{ID: "note:a.md", Label: "a", Note: "a.md"},
			{ID: "note:b.md", Label: "b", Note: "b.md"},
			{ID: "note:c.md", Label: "c", Note: "c.md"},
			{ID: "tag:billing", Label: "billing", Kind: KindTag},
			{ID: "entity:x", Label: "x", Kind: "entity"},
		},
		Edges: []*Edge{
			{From: "note:a.md", To: "tag:billing", Relation: RelTagged, Provenance: ProvExtracted, Confidence: 1},
			{From: "note:b.md", To: "tag:billing", Relation: RelTagged, Provenance: ProvExtracted, Confidence: 1},
			{From: "note:c.md", To: "tag:billing", Relation: RelTagged, Provenance: ProvExtracted, Confidence: 1},
			{From: "note:a.md", To: "entity:x", Relation: "mentions", Provenance: ProvExtracted, Confidence: 1},
		},
	}
	nb := testIndex(g).Neighbours("note:a.md")
	got := map[string]string{}
	for _, n := range nb {
		got[n.Node.ID] = n.Relation
	}
	if len(nb) != 2 || got["note:b.md"] != RelTagged || got["note:c.md"] != RelTagged {
		t.Fatalf("want tag-expanded b and c only, got %+v", nb)
	}
	if testIndex(g).Node("note:a.md") == nil || testIndex(g).Node("missing") != nil {
		t.Errorf("Node lookup wrong")
	}
	if got := (*Index)(nil).Neighbours("x"); got != nil {
		t.Errorf("nil index should list nothing")
	}
}
