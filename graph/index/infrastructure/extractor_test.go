package infrastructure

import (
	"reflect"
	"sort"
	"testing"
)

func TestLinkReferenceExtractor_MarkdownLinks(t *testing.T) {
	extr := NewLinkReferenceExtractor("/root")
	content := "See [other](other.md) and [nested](sub/nested.md)."
	got := extr.Extract(content, "/root/index.md")
	want := []string{"/root/other.md", "/root/sub/nested.md"}
	assertPaths(t, got, want)
}

func TestLinkReferenceExtractor_Wikilinks(t *testing.T) {
	extr := NewLinkReferenceExtractor("/root")
	content := "See [[other]] and [[sub/nested|Nested]]."
	got := extr.Extract(content, "/root/index.md")
	want := []string{"/root/other", "/root/sub/nested"}
	assertPaths(t, got, want)
}

func TestLinkReferenceExtractor_DropsExternalURLs(t *testing.T) {
	extr := NewLinkReferenceExtractor("/root")
	content := "[ext](https://example.com/page) [mail](mailto:a@b.com)"
	got := extr.Extract(content, "/root/index.md")
	if len(got) != 0 {
		t.Fatalf("expected no references, got %v", got)
	}
}

func TestLinkReferenceExtractor_DropsOutsideRoot(t *testing.T) {
	extr := NewLinkReferenceExtractor("/root")
	content := "[outside](../outside.md)"
	got := extr.Extract(content, "/root/index.md")
	if len(got) != 0 {
		t.Fatalf("expected no references, got %v", got)
	}
}

func TestFrontmatterTopicExtractor_Topics(t *testing.T) {
	extr := NewFrontmatterTopicExtractor()
	content := "---\ntopics:\n  - go\n  - graphs\n---\n\nBody text."
	got := extr.Extract(content)
	want := []string{"go", "graphs"}
	assertPaths(t, got, want)
}

func TestFrontmatterTopicExtractor_Tags(t *testing.T) {
	extr := NewFrontmatterTopicExtractor()
	content := "---\ntags: [alpha, beta]\n---\n\nBody text."
	got := extr.Extract(content)
	want := []string{"alpha", "beta"}
	assertPaths(t, got, want)
}

func TestFrontmatterTopicExtractor_NoFrontmatter(t *testing.T) {
	extr := NewFrontmatterTopicExtractor()
	got := extr.Extract("Just a plain file with no frontmatter.")
	if got != nil {
		t.Fatalf("expected nil, got %v", got)
	}
}

func assertPaths(t *testing.T, got, want []string) {
	t.Helper()
	sort.Strings(got)
	sort.Strings(want)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}
