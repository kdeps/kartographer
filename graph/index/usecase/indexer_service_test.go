package usecase

import (
	"path/filepath"
	"sort"
	"testing"

	"github.com/spf13/afero"

	"github.com/kdeps/kartographer/graph/index/infrastructure"
)

func newTestService(t *testing.T, fs afero.Fs, root string) *IndexerService {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "index.db")
	store, err := infrastructure.NewBoltIndexStore(dbPath)
	if err != nil {
		t.Fatalf("NewBoltIndexStore: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	walker := infrastructure.NewAferoFileWalker(fs)
	refExtr := infrastructure.NewLinkReferenceExtractor(root)
	topicExtr := infrastructure.NewFrontmatterTopicExtractor()
	return NewIndexerService(fs, walker, refExtr, topicExtr, store)
}

func TestIndexerService_IndexFolder_BuildsReferenceGraph(t *testing.T) {
	fs := afero.NewMemMapFs()
	must(t, afero.WriteFile(fs, "/root/a.md",
		[]byte("---\ntopics: [go]\n---\nSee [b](b.md)."), 0o644))
	must(t, afero.WriteFile(fs, "/root/b.md",
		[]byte("---\ntopics: [go]\n---\nNo links here."), 0o644))
	must(t, afero.WriteFile(fs, "/root/c.md",
		[]byte("---\ntopics: [rust]\n---\nUnrelated."), 0o644))

	svc := newTestService(t, fs, "/root")
	if n, err := svc.IndexFolder("/root", []string{".md"}); err != nil {
		t.Fatalf("IndexFolder: %v", err)
	} else if n != 3 {
		t.Fatalf("IndexFolder: got %d files indexed, want 3", n)
	}

	refGraph, err := svc.BuildReferenceGraph()
	if err != nil {
		t.Fatalf("BuildReferenceGraph: %v", err)
	}
	if len(refGraph["/root/a.md"]) != 1 || refGraph["/root/a.md"][0] != "/root/b.md" {
		t.Fatalf("unexpected references for a.md: %v", refGraph["/root/a.md"])
	}

	files, err := svc.FilesForTopic("go")
	if err != nil {
		t.Fatalf("FilesForTopic: %v", err)
	}
	sort.Strings(files)
	want := []string{"/root/a.md", "/root/b.md"}
	if len(files) != len(want) || files[0] != want[0] || files[1] != want[1] {
		t.Fatalf("got %v, want %v", files, want)
	}
}

func TestIndexerService_FilesRelatedByTopic(t *testing.T) {
	fs := afero.NewMemMapFs()
	must(t, afero.WriteFile(fs, "/root/a.md", []byte("---\ntopics: [go]\n---\nA"), 0o644))
	must(t, afero.WriteFile(fs, "/root/b.md", []byte("---\ntopics: [go]\n---\nB"), 0o644))
	must(t, afero.WriteFile(fs, "/root/c.md", []byte("No frontmatter."), 0o644))

	svc := newTestService(t, fs, "/root")
	if _, err := svc.IndexFolder("/root", []string{".md"}); err != nil {
		t.Fatalf("IndexFolder: %v", err)
	}

	related, err := svc.FilesRelatedByTopic("/root/a.md")
	if err != nil {
		t.Fatalf("FilesRelatedByTopic: %v", err)
	}
	if len(related) != 1 || related[0] != "/root/b.md" {
		t.Fatalf("got %v, want [/root/b.md]", related)
	}

	related, err = svc.FilesRelatedByTopic("/root/c.md")
	if err != nil {
		t.Fatalf("FilesRelatedByTopic: %v", err)
	}
	if len(related) != 0 {
		t.Fatalf("expected no related files for c.md, got %v", related)
	}
}

func TestIndexerService_IndexFolder_SetKnownFilesForNonRelativeResolution(t *testing.T) {
	fs := afero.NewMemMapFs()
	must(t, afero.WriteFile(fs, "/root/main.go", []byte(`package main

import "example.com/root/pkgname"

func main() {}
`), 0o644))
	// Named differently from its directory so the known-file index's
	// directory-basename entry ("pkgname" -> the dir) stays unambiguous.
	must(t, afero.WriteFile(fs, "/root/pkgname/impl.go", []byte("package pkgname\n"), 0o644))

	dbPath := filepath.Join(t.TempDir(), "index.db")
	store, err := infrastructure.NewBoltIndexStore(dbPath)
	if err != nil {
		t.Fatalf("NewBoltIndexStore: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	walker := infrastructure.NewAferoFileWalker(fs)
	refExtr := infrastructure.NewCompositeReferenceExtractor("/root")
	topicExtr := infrastructure.NewCompositeTopicExtractor()
	svc := NewIndexerService(fs, walker, refExtr, topicExtr, store)

	if _, err := svc.IndexFolder("/root", []string{".go"}); err != nil {
		t.Fatalf("IndexFolder: %v", err)
	}

	refGraph, err := svc.BuildReferenceGraph()
	if err != nil {
		t.Fatalf("BuildReferenceGraph: %v", err)
	}
	got := refGraph["/root/main.go"]
	if len(got) != 1 || got[0] != "/root/pkgname" {
		t.Fatalf("expected main.go's non-relative import resolved via the known file set to /root/pkgname, got %v", got)
	}
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
