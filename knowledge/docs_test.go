package knowledge

import (
	"slices"
	"testing"
)

func TestResolver_WikiAndRelative(t *testing.T) {
	docs := []Doc{
		{Path: "shared/Pricing.md", Title: "Pricing", Aliases: []string{"rates"}},
		{Path: "projects/a/Pricing.md", Title: "Pricing"},
		{Path: "projects/a/notes/Stripe.md", Title: "Stripe"},
	}
	r := NewResolver(docs)
	if got := r.Wiki("shared/pricing", "x.md"); got != "shared/Pricing.md" {
		t.Errorf("path target = %q", got)
	}
	if got := r.Wiki("Pricing", "projects/a/app.md"); got != "projects/a/Pricing.md" {
		t.Errorf("nearest title = %q", got)
	}
	if got := r.Wiki("rates", "x.md"); got != "shared/Pricing.md" {
		t.Errorf("alias = %q", got)
	}
	if got := r.Wiki("missing", "x.md"); got != "" {
		t.Errorf("unknown target = %q", got)
	}
	if got := r.Relative("../Pricing.md", "projects/a/notes/Stripe.md"); got != "projects/a/Pricing.md" {
		t.Errorf("relative = %q", got)
	}
}

func TestAddDoc_LinksTagsAndMissing(t *testing.T) {
	docs := []Doc{
		{Path: "hub.md", Title: "Hub"},
		{Path: "spoke.md", Title: "Spoke"},
	}
	r := NewResolver(docs)
	b := NewBuilder()
	missing := b.AddDoc(Doc{
		Path: "hub.md", Title: "Hub", Tags: []string{"Billing"},
		Links: []string{"Spoke", "Nowhere"}, LinkLine: map[string]int{"Spoke": 3, "Nowhere": 4},
		MetaLinks: []string{"Spoke"}, MetaLine: map[string]int{"Spoke": 1},
		TagLine: map[string]int{"Billing": 2},
	}, r)
	if !slices.Equal(missing, []string{"Nowhere"}) {
		t.Fatalf("missing = %v", missing)
	}
	g := b.Graph()
	got := map[string]string{}
	for _, e := range g.Edges {
		got[e.From+" -> "+e.To+" "+e.Relation] = e.SourceLocation
	}
	want := map[string]string{
		"note:hub.md -> note:spoke.md": RelReferences + "@L3",
		"note:hub.md -> tag:billing":   RelTagged + "@L2",
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("edge %s = %q, want %q (all: %v)", k, got[k], v, got)
		}
	}
}

func TestLineRef(t *testing.T) {
	if LineRef(0) != "" || LineRef(7) != "L7" {
		t.Errorf("LineRef wrong")
	}
}
