package notes

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFilesAndManifest(t *testing.T) {
	root := t.TempDir()
	write := func(rel, text string) {
		p := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(text), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write("b.md", "b")
	write("a/One.MD", "one")
	write(".hidden/x.md", "x")
	write("a/.secret.md", "x")
	write("big.md", strings.Repeat("x", 50))
	write("report.md", "generated")
	write("note.txt", "not a note")

	files, err := Files(root, FileOptions{Ext: ".md", MaxSize: 10,
		Skip: func(rel, _ string) bool { return rel == "report.md" }})
	if err != nil {
		t.Fatal(err)
	}
	var rels []string
	for _, f := range files {
		rels = append(rels, f.Rel)
	}
	if strings.Join(rels, ",") != "a/One.MD,b.md" {
		t.Fatalf("files = %v", rels)
	}
	m := ManifestOf(files)
	if !m.Equal(ManifestOf(files)) || m["b.md"].Size != 1 {
		t.Errorf("manifest = %v", m)
	}
	write("b.md", "changed")
	again, _ := Files(root, FileOptions{Ext: ".md", MaxSize: 10})
	if m.Equal(ManifestOf(again)) {
		t.Error("a changed file changes the manifest")
	}
	if _, err := Files(filepath.Join(root, "missing"), FileOptions{Ext: ".md"}); err == nil {
		t.Error("a missing root is an error")
	}
}
