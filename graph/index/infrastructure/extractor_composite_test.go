package infrastructure

import "testing"

func TestCompositeReferenceExtractor_Markdown(t *testing.T) {
	extr := NewCompositeReferenceExtractor("/root")
	got := extr.Extract("[other](other.md)", "/root/index.md")
	assertPaths(t, got, []string{"/root/other.md"})
}

func TestCompositeReferenceExtractor_Text(t *testing.T) {
	extr := NewCompositeReferenceExtractor("/root")
	got := extr.Extract("[other](other.txt)", "/root/index.txt")
	assertPaths(t, got, []string{"/root/other.txt"})
}

func TestCompositeReferenceExtractor_HTML(t *testing.T) {
	extr := NewCompositeReferenceExtractor("/root")
	got := extr.Extract(`<a href="other.html">x</a>`, "/root/index.html")
	assertPaths(t, got, []string{"/root/other.html"})
}

func TestCompositeReferenceExtractor_SourceCode(t *testing.T) {
	extr := NewCompositeReferenceExtractor("/root")
	if fsa, ok := extr.(interface{ SetKnownFiles([]string) }); ok {
		fsa.SetKnownFiles([]string{"/root/util.rb", "/root/main.rb"})
	} else {
		t.Fatal("expected CompositeReferenceExtractor to implement FileSetAware")
	}
	got := extr.Extract(`require_relative 'util'`, "/root/main.rb")
	assertPaths(t, got, []string{"/root/util.rb"})
}

func TestCompositeReferenceExtractor_UnknownExtension(t *testing.T) {
	extr := NewCompositeReferenceExtractor("/root")
	got := extr.Extract(`{"topics": ["go"]}`, "/root/data.json")
	if len(got) != 0 {
		t.Fatalf("expected no references for a bare JSON file, got %v", got)
	}
}

func TestCompositeTopicExtractor_Frontmatter(t *testing.T) {
	extr := NewCompositeTopicExtractor()
	got := extr.Extract("---\ntopics: [go]\n---\nBody.", "/root/index.md")
	assertPaths(t, got, []string{"go"})
}

func TestCompositeTopicExtractor_BareYAML(t *testing.T) {
	extr := NewCompositeTopicExtractor()
	got := extr.Extract("topics:\n  - go\n  - graphs\n", "/root/data.yaml")
	assertPaths(t, got, []string{"go", "graphs"})
}

func TestCompositeTopicExtractor_BareJSON(t *testing.T) {
	extr := NewCompositeTopicExtractor()
	got := extr.Extract(`{"tags": ["alpha", "beta"]}`, "/root/data.json")
	assertPaths(t, got, []string{"alpha", "beta"})
}

func TestCompositeTopicExtractor_NonTopicExtensionIgnored(t *testing.T) {
	extr := NewCompositeTopicExtractor()
	// Valid YAML-parseable content, but a .go extension isn't in
	// topicExtensions -- must not be scanned for a bare top-level topics key.
	got := extr.Extract("topics: [go]", "/root/main.go")
	if got != nil {
		t.Fatalf("expected nil for a non-topic extension, got %v", got)
	}
}

func TestCompositeTopicExtractor_BareYAMLNoTopicsKey(t *testing.T) {
	extr := NewCompositeTopicExtractor()
	got := extr.Extract("name: config\nvalue: 1\n", "/root/data.yaml")
	if got != nil {
		t.Fatalf("expected nil when no topics/tags key present, got %v", got)
	}
}
