package knowledge

import (
	"testing"
)

var testParse = ParseOptions{Kinds: []string{"tool", "decision"}, DefaultKind: "concept", MaxNameLen: 20}

func TestParseFound(t *testing.T) {
	reply := `Sure! {"entities":[{"name":" Stripe ","kind":"Tool"},{"name":"Monthly","kind":"plan"},` +
		`{"name":"` + "a-very-long-entity-name-indeed" + `","kind":"tool"},{"name":"","kind":"tool"}],` +
		`"relations":[{"from":"Stripe","to":"Monthly","relation":"Depends_On","confidence":1.7},` +
		`{"from":"Stripe","to":"stripe","relation":"x","confidence":0.5},{"from":"Stripe","to":"Gone","relation":"x"}]} trailing`
	f, err := ParseFound(reply, testParse)
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Entities) != 2 || f.Entities[0] != (Entity{"Stripe", "tool"}) || f.Entities[1].Kind != "concept" {
		t.Errorf("entities = %+v", f.Entities)
	}
	if len(f.Relations) != 1 || f.Relations[0].Relation != "depends_on" || f.Relations[0].Confidence != 1 {
		t.Errorf("relations = %+v", f.Relations)
	}
	if _, err := ParseFound("no json here", testParse); err == nil {
		t.Error("a reply without JSON fails")
	}
	cut, err := ParseFound(`{"entities":[{"name":"A","kind":"tool"},{"name":"B","kin`, testParse)
	if err != nil || len(cut.Entities) != 1 {
		t.Errorf("a cut-off reply is salvaged: %+v, %v", cut, err)
	}
	if _, err := ParseFound(`{"entities": [`, testParse); err == nil {
		t.Error("nothing to salvage fails")
	}
}

func TestAddFound(t *testing.T) {
	r := NewResolver([]Doc{{Path: "Stripe.md", Title: "Stripe"}})
	b := NewBuilder()
	b.AddFound("a.md", Found{
		Entities: []Entity{{"Stripe", "tool"}, {"Monthly", "decision"}},
		Relations: []Relation{
			{From: "Stripe", To: "Monthly", Relation: "depends_on", Confidence: 0.9},
			{From: "Monthly", To: "Stripe", Relation: "invented", Confidence: 0.9},
			{From: "Monthly", To: "Gone", Relation: "depends_on", Confidence: 0.9},
		},
	}, r, MergeOptions{
		Mention: "mentions", MentionConfidence: 0.9, Relations: []string{"depends_on"},
		Fallback: "relates_to", FallbackMax: 0.49, AmbiguousBelow: 0.5,
		Evidence: func(names ...string) string { return "L1" },
	})
	got := map[string]*Edge{}
	for _, e := range b.Graph().Edges {
		got[e.From+" "+e.Relation+" "+e.To] = e
	}
	if e := got["note:a.md mentions note:Stripe.md"]; e == nil || e.Provenance != ProvInferred || e.SourceLocation != "L1" {
		t.Errorf("a named note is the entity: %+v", got)
	}
	if got["note:a.md mentions entity:monthly"] == nil {
		t.Errorf("an unnamed entity gets its own node: %+v", got)
	}
	if e := got["note:Stripe.md depends_on entity:monthly"]; e == nil || e.Provenance != ProvInferred {
		t.Errorf("a listed relation is kept: %+v", got)
	}
	if e := got["entity:monthly relates_to note:Stripe.md"]; e == nil || e.Provenance != ProvAmbiguous || e.Confidence != 0.49 {
		t.Errorf("an unlisted relation falls back, capped and ambiguous: %+v", e)
	}
	if len(got) != 4 {
		t.Errorf("a relation to an unknown entity is dropped: %d edges", len(got))
	}
}
