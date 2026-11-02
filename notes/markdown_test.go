package notes

import (
	"reflect"
	"testing"
	"time"
)

func TestParseMarkdown(t *testing.T) {
	data := []byte("---\ntags: [billing, \"#ops\"]\nlinks: [\"[[Pricing]]\", Tiers]\nsummary: s\n---\n" +
		"See [[Stripe]] and [the plan](plan.md).\n\n```\n[[Not a link]]\n```\nInline `[[code]]` too.\n")
	m := ParseMarkdown(data, FrontmatterKeys{Tags: "tags", Links: "links"})
	if !reflect.DeepEqual(m.Tags, []string{"billing", "ops"}) {
		t.Errorf("tags = %v", m.Tags)
	}
	if !reflect.DeepEqual(m.MetaLinks, []string{"Pricing", "Tiers"}) {
		t.Errorf("meta links = %v", m.MetaLinks)
	}
	if !reflect.DeepEqual(m.Links, []string{"Stripe"}) {
		t.Errorf("code must not hold links: %v", m.Links)
	}
	if !reflect.DeepEqual(m.MDLinks, []string{"plan.md"}) {
		t.Errorf("md links = %v", m.MDLinks)
	}
	if m.LinkLine["Stripe"] != 6 || m.MDLine["plan.md"] != 6 || m.MetaLine["Pricing"] != 3 || m.TagLine["billing"] != 2 {
		t.Errorf("lines: link %v md %v meta %v tag %v", m.LinkLine, m.MDLine, m.MetaLine, m.TagLine)
	}
	if MetaString(m.Meta, "summary") != "s" {
		t.Errorf("summary = %q", MetaString(m.Meta, "summary"))
	}
}

func TestSplitFrontmatter(t *testing.T) {
	meta, body := SplitFrontmatter([]byte("\ufeff---\r\na: 1\r\n---\r\nbody\r\n"))
	if meta["a"] != 1 || body != "body\r\n" {
		t.Errorf("CRLF with BOM: %v %q", meta, body)
	}
	if meta, body := SplitFrontmatter([]byte("no front matter")); len(meta) != 0 || body != "no front matter" {
		t.Errorf("plain note: %v %q", meta, body)
	}
	if meta, _ := SplitFrontmatter([]byte("---\na: [\n---\nbody")); len(meta) != 0 {
		t.Errorf("bad YAML gives empty meta: %v", meta)
	}
	if meta, body := SplitFrontmatter([]byte("---\na: 1\nno end")); len(meta) != 0 || body == "" {
		t.Errorf("an unclosed block is body: %v %q", meta, body)
	}
}

func TestMetaStringAndList(t *testing.T) {
	meta := map[string]any{
		"d": time.Date(2026, 11, 2, 0, 0, 0, 0, time.UTC), "ts": time.Date(2026, 11, 2, 3, 4, 5, 0, time.UTC),
		"n": 3, "csv": "a, #b, ,c", "list": []any{"#x", " y ", 4, ""},
	}
	for key, want := range map[string]string{"d": "2026-11-02", "ts": "2026-11-02T03:04:05Z", "n": "3", "missing": ""} {
		if got := MetaString(meta, key); got != want {
			t.Errorf("MetaString(%s) = %q, want %q", key, got, want)
		}
	}
	if got := MetaList(meta, "csv"); !reflect.DeepEqual(got, []string{"a", "b", "c"}) {
		t.Errorf("csv list = %v", got)
	}
	if got := MetaList(meta, "list"); !reflect.DeepEqual(got, []string{"x", "y"}) {
		t.Errorf("yaml list = %v", got)
	}
	if MetaList(meta, "n") != nil {
		t.Error("a number is no list")
	}
}
