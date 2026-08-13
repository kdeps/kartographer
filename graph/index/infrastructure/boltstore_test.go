package infrastructure

import (
	"path/filepath"
	"testing"

	"github.com/kdeps/kartographer/graph/index/domain"
)

func newTestStore(t *testing.T) *BoltIndexStore {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "index.db")
	store, err := NewBoltIndexStore(dbPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store
}

func TestBoltIndexStore_PutGetFile(t *testing.T) {
	store := newTestStore(t)

	rec := domain.NewFileRecord("/root/a.md", []string{"/root/b.md"}, []string{"go"}, 123)
	if err := store.PutFile(rec); err != nil {
		t.Fatalf("PutFile: %v", err)
	}

	got, found, err := store.GetFile("/root/a.md")
	if err != nil {
		t.Fatalf("GetFile: %v", err)
	}
	if !found {
		t.Fatalf("expected file to be found")
	}
	if got.Path != rec.Path || len(got.References) != 1 || len(got.Topics) != 1 {
		t.Fatalf("got %+v, want %+v", got, rec)
	}
}

func TestBoltIndexStore_GetFile_NotFound(t *testing.T) {
	store := newTestStore(t)
	_, found, err := store.GetFile("/does/not/exist.md")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found {
		t.Fatalf("expected not found")
	}
}

func TestBoltIndexStore_TopicIndexSync(t *testing.T) {
	store := newTestStore(t)

	rec := domain.NewFileRecord("/root/a.md", nil, []string{"go", "graphs"}, 1)
	if err := store.PutFile(rec); err != nil {
		t.Fatalf("PutFile: %v", err)
	}

	files, err := store.FilesByTopic("go")
	if err != nil {
		t.Fatalf("FilesByTopic: %v", err)
	}
	assertPaths(t, files, []string{"/root/a.md"})

	// Re-index the same file dropping the "go" topic: it should disappear
	// from that topic's inverted index.
	rec2 := domain.NewFileRecord("/root/a.md", nil, []string{"graphs"}, 2)
	if err := store.PutFile(rec2); err != nil {
		t.Fatalf("PutFile: %v", err)
	}

	files, err = store.FilesByTopic("go")
	if err != nil {
		t.Fatalf("FilesByTopic: %v", err)
	}
	if len(files) != 0 {
		t.Fatalf("expected topic 'go' to have no files, got %v", files)
	}

	files, err = store.FilesByTopic("graphs")
	if err != nil {
		t.Fatalf("FilesByTopic: %v", err)
	}
	assertPaths(t, files, []string{"/root/a.md"})
}

func TestBoltIndexStore_AllFiles(t *testing.T) {
	store := newTestStore(t)

	must(t, store.PutFile(domain.NewFileRecord("/root/a.md", nil, nil, 1)))
	must(t, store.PutFile(domain.NewFileRecord("/root/b.md", nil, nil, 1)))

	recs, err := store.AllFiles()
	if err != nil {
		t.Fatalf("AllFiles: %v", err)
	}
	if len(recs) != 2 {
		t.Fatalf("expected 2 records, got %d", len(recs))
	}
}
