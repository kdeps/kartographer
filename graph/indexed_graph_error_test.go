package graph

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/log"
	"github.com/spf13/afero"

	indexdomain "github.com/kdeps/kartographer/graph/index/domain"
	"github.com/kdeps/kartographer/graph/index/usecase"
)

// mockStore is an indexdomain.IndexStore whose methods can be made to fail
// individually, isolating GraphFile/GraphTopic/GraphAll error branches that
// share an underlying bbolt db in the real implementation and so can't be
// triggered independently there.
type mockStore struct {
	getFile               *indexdomain.FileRecord
	getFileFound          bool
	getFileErr            error
	allFilesErr           error
	filesByTopic          map[string][]string
	filesByTopicErrOnCall int
	filesByTopicCalls     int
}

func (m *mockStore) PutFile(*indexdomain.FileRecord) error { return nil }

func (m *mockStore) GetFile(string) (*indexdomain.FileRecord, bool, error) {
	if m.getFileErr != nil {
		return nil, false, m.getFileErr
	}
	return m.getFile, m.getFileFound, nil
}

func (m *mockStore) AllFiles() ([]*indexdomain.FileRecord, error) {
	if m.allFilesErr != nil {
		return nil, m.allFilesErr
	}
	return nil, nil
}

func (m *mockStore) FilesByTopic(topic string) ([]string, error) {
	m.filesByTopicCalls++
	if m.filesByTopicErrOnCall != 0 && m.filesByTopicCalls == m.filesByTopicErrOnCall {
		return nil, errors.New("files by topic failed")
	}
	return m.filesByTopic[topic], nil
}

func (m *mockStore) Close() error { return nil }

func newMockIndexedGraph(store indexdomain.IndexStore) *IndexedGraph {
	ig := &IndexedGraph{Fs: afero.NewMemMapFs(), Logger: log.Default()}
	ig.idx = usecase.NewIndexerService(ig.Fs, nil, nil, nil, store)
	return ig
}

func TestNewIndexedGraph_OpenError(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "graph.db")
	if err := os.WriteFile(dbPath, []byte("not a bbolt database"), 0o600); err != nil {
		t.Fatalf("write bogus db: %v", err)
	}
	if _, err := NewIndexedGraph(afero.NewOsFs(), log.Default(), dbPath); err == nil {
		t.Fatal("expected bbolt open error")
	}
}

func TestIndexedGraph_Indexer_LazyInit(t *testing.T) {
	fs := afero.NewMemMapFs()
	ig := newTestIndexedGraph(t, fs)
	// No IndexFolder call: ig.idx is nil, so GraphAll must lazily build one.
	references, roots, err := ig.GraphAll()
	if err != nil {
		t.Fatalf("GraphAll: %v", err)
	}
	if len(references) != 0 || len(roots) != 0 {
		t.Fatalf("expected empty graph on unindexed store, got refs=%v roots=%v", references, roots)
	}
}

func TestIndexedGraph_IndexFolder_DefaultExtensions(t *testing.T) {
	fs := afero.NewMemMapFs()
	seedFixture(t, fs)

	ig := newTestIndexedGraph(t, fs)
	n, err := ig.IndexFolder("/root", nil)
	if err != nil {
		t.Fatalf("IndexFolder: %v", err)
	}
	if n != 3 {
		t.Fatalf("IndexFolder with default extensions: got %d, want 3", n)
	}
}

func TestIndexedGraph_GraphFile_BuildReferenceGraphError(t *testing.T) {
	ig := newMockIndexedGraph(&mockStore{allFilesErr: errors.New("all files failed")})
	if _, _, err := ig.GraphFile("/root/a.md"); err == nil {
		t.Fatal("expected BuildReferenceGraph error")
	}
}

func TestIndexedGraph_GraphFile_RelatedByTopicError(t *testing.T) {
	ig := newMockIndexedGraph(&mockStore{getFileErr: errors.New("get file failed")})
	if _, _, err := ig.GraphFile("/root/a.md"); err == nil {
		t.Fatal("expected FilesRelatedByTopic error")
	}
}

func TestIndexedGraph_GraphTopic_FilesForTopicError(t *testing.T) {
	ig := newMockIndexedGraph(&mockStore{filesByTopicErrOnCall: 1})
	if _, _, err := ig.GraphTopic("go"); err == nil {
		t.Fatal("expected FilesForTopic error")
	}
}

func TestIndexedGraph_GraphTopic_BuildReferenceGraphError(t *testing.T) {
	ig := newMockIndexedGraph(&mockStore{allFilesErr: errors.New("all files failed")})
	if _, _, err := ig.GraphTopic("go"); err == nil {
		t.Fatal("expected BuildReferenceGraph error")
	}
}

func TestIndexedGraph_GraphAll_BuildReferenceGraphError(t *testing.T) {
	ig := newMockIndexedGraph(&mockStore{allFilesErr: errors.New("all files failed")})
	if _, _, err := ig.GraphAll(); err == nil {
		t.Fatal("expected BuildReferenceGraph error")
	}
}

func TestIndexedGraph_ShowFile_Error(t *testing.T) {
	ig := newMockIndexedGraph(&mockStore{allFilesErr: errors.New("all files failed")})
	if err := ig.ShowFile("/root/a.md"); err == nil {
		t.Fatal("expected ShowFile to propagate GraphFile error")
	}
}

func TestIndexedGraph_ShowFile_NoRelated(t *testing.T) {
	fs := afero.NewMemMapFs()
	if err := afero.WriteFile(fs, "/root/solo.md", []byte("No frontmatter, no links."), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	ig := newTestIndexedGraph(t, fs)
	if _, err := ig.IndexFolder("/root", []string{".md"}); err != nil {
		t.Fatalf("IndexFolder: %v", err)
	}

	out := captureOutput(func() {
		if err := ig.ShowFile("/root/solo.md"); err != nil {
			t.Fatalf("ShowFile: %v", err)
		}
	})
	if !strings.Contains(out, "(none)") {
		t.Fatalf("expected '(none)' for a file with no related-by-topic files, got:\n%s", out)
	}
}

func TestIndexedGraph_ShowTopic_Error(t *testing.T) {
	ig := newMockIndexedGraph(&mockStore{filesByTopicErrOnCall: 1})
	if err := ig.ShowTopic("go"); err == nil {
		t.Fatal("expected ShowTopic to propagate GraphTopic error")
	}
}

func TestIndexedGraph_ShowAll_Error(t *testing.T) {
	ig := newMockIndexedGraph(&mockStore{allFilesErr: errors.New("all files failed")})
	if err := ig.ShowAll(); err == nil {
		t.Fatal("expected ShowAll to propagate GraphAll error")
	}
}
