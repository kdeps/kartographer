package infrastructure

import (
	"sort"
	"testing"

	"github.com/spf13/afero"
)

func TestAferoFileWalker_FiltersByExtension(t *testing.T) {
	fs := afero.NewMemMapFs()
	must(t, afero.WriteFile(fs, "/root/a.md", []byte("a"), 0o644))
	must(t, afero.WriteFile(fs, "/root/b.txt", []byte("b"), 0o644))
	must(t, afero.WriteFile(fs, "/root/skip.png", []byte("x"), 0o644))
	must(t, afero.WriteFile(fs, "/root/sub/c.md", []byte("c"), 0o644))

	w := NewAferoFileWalker(fs)
	got, err := w.Walk("/root", []string{".md", ".txt"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sort.Strings(got)
	want := []string{"/root/a.md", "/root/b.txt", "/root/sub/c.md"}
	assertPaths(t, got, want)
}

func TestAferoFileWalker_NoMatches(t *testing.T) {
	fs := afero.NewMemMapFs()
	must(t, afero.WriteFile(fs, "/root/a.png", []byte("a"), 0o644))

	w := NewAferoFileWalker(fs)
	got, err := w.Walk("/root", []string{".md"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected no matches, got %v", got)
	}
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
