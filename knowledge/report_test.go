package knowledge

import "testing"

func TestNewReport_RanksGroupsBridgesAndTruncates(t *testing.T) {
	g := &Graph{
		Nodes: []*Node{
			{ID: "note:a.md", Label: "a", Note: "a.md", Community: 0, Summary: "alpha"},
			{ID: "note:b.md", Label: "b", Note: "b.md", Community: 0},
			{ID: "note:c.md", Label: "c", Note: "c.md", Community: 1},
			{ID: "note:d.md", Label: "d", Note: "d.md", Community: 1},
			{ID: "note:lone.md", Label: "lone", Note: "lone.md", Community: 2},
			{ID: "tag:x", Label: "#x", Kind: KindTag, Community: 0},
		},
		Edges: []*Edge{
			{From: "note:a.md", To: "note:b.md", Provenance: ProvExtracted, Confidence: 1},
			{From: "note:c.md", To: "note:d.md", Provenance: ProvExtracted, Confidence: 1},
			{From: "note:b.md", To: "note:c.md", Provenance: ProvInferred, Confidence: 0.6},
			{From: "note:a.md", To: "note:lone.md", Provenance: ProvAmbiguous, Confidence: 0.9},
			{From: "note:b.md", To: "note:lone.md", Provenance: ProvAmbiguous, Confidence: 0.5},
			{From: "note:c.md", To: "note:lone.md", Provenance: ProvAmbiguous, Confidence: 0.4},
		},
		Missing: []string{"m1", "m2", "m3"},
	}
	r := NewReport(g, 0.5, ReportOptions{Top: 2, CommunityTop: 1, MissingTop: 2})

	if r.Notes != 5 || r.Nodes != 6 || r.Edges != 6 || r.Extracted != 2 || r.Communities != 3 {
		t.Errorf("counts wrong: %+v", r)
	}
	if len(r.GodNodes) != 2 || r.GodNodes[0].Node.Kind == KindTag {
		t.Errorf("god nodes must skip tags and be capped: %+v", r.GodNodes)
	}
	if len(r.Groups) != 2 {
		t.Errorf("single-node community must be left out: %+v", r.Groups)
	}
	if r.Groups[0].Summary != "alpha" || len(r.Groups[0].Top) != 1 {
		t.Errorf("group summary or top list wrong: %+v", r.Groups[0])
	}
	if len(r.Review) != 2 || r.Review[0].Confidence != 0.9 {
		t.Errorf("review must be most confident first and capped: %+v", r.Review)
	}
	if len(r.Missing) != 2 {
		t.Errorf("missing must be capped: %v", r.Missing)
	}
	if len(r.Surprising) != 1 || r.Surprising[0].Edge.From != "note:b.md" || r.Surprising[0].Edge.To != "note:c.md" {
		t.Errorf("the lone link between communities 0 and 1 is the surprise: %+v", r.Surprising)
	}
}
