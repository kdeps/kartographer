package graph

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/log"
	"github.com/spf13/afero"
)

func newTestIndexedGraph(t *testing.T, fs afero.Fs) *IndexedGraph {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "index.db")
	ig, err := NewIndexedGraph(fs, log.Default(), dbPath)
	if err != nil {
		t.Fatalf("NewIndexedGraph: %v", err)
	}
	t.Cleanup(func() { _ = ig.Close() })
	return ig
}

func seedFixture(t *testing.T, fs afero.Fs) {
	t.Helper()
	files := map[string]string{
		"/root/a.md": "---\ntopics: [go]\n---\nSee [b](b.md).",
		"/root/b.md": "---\ntopics: [go]\n---\nNo links.",
		"/root/c.md": "---\ntopics: [rust]\n---\nUnrelated.",
	}
	for path, content := range files {
		if err := afero.WriteFile(fs, path, []byte(content), 0o644); err != nil {
			t.Fatalf("WriteFile %s: %v", path, err)
		}
	}
}

func TestIndexedGraph_IndexFolder_Count(t *testing.T) {
	fs := afero.NewMemMapFs()
	seedFixture(t, fs)

	ig := newTestIndexedGraph(t, fs)
	n, err := ig.IndexFolder("/root", []string{".md"})
	if err != nil {
		t.Fatalf("IndexFolder: %v", err)
	}
	if n != 3 {
		t.Fatalf("IndexFolder: got %d files indexed, want 3", n)
	}
}

func TestIndexedGraph_GraphFile(t *testing.T) {
	fs := afero.NewMemMapFs()
	seedFixture(t, fs)

	ig := newTestIndexedGraph(t, fs)
	if _, err := ig.IndexFolder("/root", []string{".md"}); err != nil {
		t.Fatalf("IndexFolder: %v", err)
	}

	references, related, err := ig.GraphFile("/root/a.md")
	if err != nil {
		t.Fatalf("GraphFile: %v", err)
	}
	if len(references["/root/a.md"]) != 1 || references["/root/a.md"][0] != "/root/b.md" {
		t.Fatalf("unexpected references for a.md: %v", references["/root/a.md"])
	}
	if len(related) != 1 || related[0] != "/root/b.md" {
		t.Fatalf("unexpected related-by-topic for a.md: %v", related)
	}
}

func TestIndexedGraph_GraphTopic(t *testing.T) {
	fs := afero.NewMemMapFs()
	seedFixture(t, fs)

	ig := newTestIndexedGraph(t, fs)
	if _, err := ig.IndexFolder("/root", []string{".md"}); err != nil {
		t.Fatalf("IndexFolder: %v", err)
	}

	files, references, err := ig.GraphTopic("go")
	if err != nil {
		t.Fatalf("GraphTopic: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("unexpected files for topic go: %v", files)
	}
	if len(references) != 3 {
		t.Fatalf("unexpected reference graph size: %v", references)
	}
}

func TestIndexedGraph_GraphAll(t *testing.T) {
	fs := afero.NewMemMapFs()
	seedFixture(t, fs)

	ig := newTestIndexedGraph(t, fs)
	if _, err := ig.IndexFolder("/root", []string{".md"}); err != nil {
		t.Fatalf("IndexFolder: %v", err)
	}

	references, roots, err := ig.GraphAll()
	if err != nil {
		t.Fatalf("GraphAll: %v", err)
	}
	if len(references) != 3 {
		t.Fatalf("unexpected reference graph size: %v", references)
	}
	// b.md is referenced by a.md, so only a.md and c.md are roots.
	want := []string{"/root/a.md", "/root/c.md"}
	if len(roots) != len(want) || roots[0] != want[0] || roots[1] != want[1] {
		t.Fatalf("got roots %v, want %v", roots, want)
	}
}

func TestIndexedGraph_ShowFile(t *testing.T) {
	fs := afero.NewMemMapFs()
	seedFixture(t, fs)

	ig := newTestIndexedGraph(t, fs)
	if _, err := ig.IndexFolder("/root", []string{".md"}); err != nil {
		t.Fatalf("IndexFolder: %v", err)
	}

	out := captureOutput(func() {
		if err := ig.ShowFile("/root/a.md"); err != nil {
			t.Fatalf("ShowFile: %v", err)
		}
	})

	if !strings.Contains(out, "/root/b.md") {
		t.Fatalf("expected reference to b.md in output, got:\n%s", out)
	}
	if !strings.Contains(out, "Related by topic:") || !strings.Contains(out, "/root/b.md") {
		t.Fatalf("expected related-by-topic section, got:\n%s", out)
	}
}

func TestIndexedGraph_ShowTopic(t *testing.T) {
	fs := afero.NewMemMapFs()
	seedFixture(t, fs)

	ig := newTestIndexedGraph(t, fs)
	if _, err := ig.IndexFolder("/root", []string{".md"}); err != nil {
		t.Fatalf("IndexFolder: %v", err)
	}

	out := captureOutput(func() {
		if err := ig.ShowTopic("go"); err != nil {
			t.Fatalf("ShowTopic: %v", err)
		}
	})

	if !strings.Contains(out, "/root/a.md") || !strings.Contains(out, "/root/b.md") {
		t.Fatalf("expected both go-tagged files in output, got:\n%s", out)
	}
	if strings.Contains(out, "/root/c.md") {
		t.Fatalf("did not expect rust-tagged file in output, got:\n%s", out)
	}
}

func TestIndexedGraph_ShowAll(t *testing.T) {
	fs := afero.NewMemMapFs()
	seedFixture(t, fs)

	ig := newTestIndexedGraph(t, fs)
	if _, err := ig.IndexFolder("/root", []string{".md"}); err != nil {
		t.Fatalf("IndexFolder: %v", err)
	}

	out := captureOutput(func() {
		if err := ig.ShowAll(); err != nil {
			t.Fatalf("ShowAll: %v", err)
		}
	})

	for _, path := range []string{"/root/a.md", "/root/b.md", "/root/c.md"} {
		if !strings.Contains(out, path) {
			t.Fatalf("expected %s in output, got:\n%s", path, out)
		}
	}
}
