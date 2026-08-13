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

func TestIndexedGraph_ShowFile(t *testing.T) {
	fs := afero.NewMemMapFs()
	seedFixture(t, fs)

	ig := newTestIndexedGraph(t, fs)
	if err := ig.IndexFolder("/root", []string{".md"}); err != nil {
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
	if err := ig.IndexFolder("/root", []string{".md"}); err != nil {
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
	if err := ig.IndexFolder("/root", []string{".md"}); err != nil {
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
