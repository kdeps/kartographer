package usecase

import (
	"errors"
	"os"
	"testing"

	"github.com/spf13/afero"

	"github.com/kdeps/kartographer/graph/index/domain"
)

// mockWalker returns a canned path list/error, letting tests force IndexFolder
// down paths a real walker wouldn't naturally take (walk failure, or listing
// a path the filesystem can't actually read).
type mockWalker struct {
	paths []string
	err   error
}

func (m *mockWalker) Walk(string, []string) ([]string, error) {
	return m.paths, m.err
}

// mockStore is a domain.IndexStore whose methods can be made to fail
// individually, isolating error branches that share an underlying bbolt db
// in the real implementation and so can't be triggered independently there.
type mockStore struct {
	getFile               *domain.FileRecord
	getFileFound          bool
	getFileErr            error
	allFilesErr           error
	filesByTopic          map[string][]string
	filesByTopicErrOnCall int
	filesByTopicCalls     int
}

func (m *mockStore) PutFile(*domain.FileRecord) error { return nil }

func (m *mockStore) GetFile(string) (*domain.FileRecord, bool, error) {
	if m.getFileErr != nil {
		return nil, false, m.getFileErr
	}
	return m.getFile, m.getFileFound, nil
}

func (m *mockStore) AllFiles() ([]*domain.FileRecord, error) {
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

func TestIndexerService_IndexFolder_WalkError(t *testing.T) {
	svc := NewIndexerService(afero.NewMemMapFs(), &mockWalker{err: errors.New("walk failed")}, nil, nil, &mockStore{})
	_, err := svc.IndexFolder("/root", nil)
	if err == nil {
		t.Fatal("expected walk error")
	}
}

func TestIndexerService_IndexFolder_ReadFileError(t *testing.T) {
	// Walker reports a path that doesn't exist on the filesystem: ReadFile
	// (inside indexFile) fails, and IndexFolder must propagate that error.
	fs := afero.NewMemMapFs()
	svc := NewIndexerService(fs, &mockWalker{paths: []string{"/root/missing.md"}}, nil, nil, &mockStore{})
	_, err := svc.IndexFolder("/root", nil)
	if err == nil {
		t.Fatal("expected indexFile/ReadFile error")
	}
}

// statFailFs wraps an afero.Fs and fails Stat for one specific path while
// leaving Open/ReadFile untouched, isolating indexFile's Stat error branch
// (which can't be reached by simply deleting the file, since that would also
// fail the preceding ReadFile).
type statFailFs struct {
	afero.Fs
	failPath string
}

func (f *statFailFs) Stat(name string) (os.FileInfo, error) {
	if name == f.failPath {
		return nil, errors.New("stat failed")
	}
	return f.Fs.Stat(name)
}

func TestIndexerService_IndexFolder_StatError(t *testing.T) {
	base := afero.NewMemMapFs()
	must(t, afero.WriteFile(base, "/root/a.md", []byte("content"), 0o644))
	fs := &statFailFs{Fs: base, failPath: "/root/a.md"}

	svc := NewIndexerService(fs, &mockWalker{paths: []string{"/root/a.md"}}, nil, nil, &mockStore{})
	_, err := svc.IndexFolder("/root", nil)
	if err == nil {
		t.Fatal("expected stat error")
	}
}

func TestIndexerService_BuildReferenceGraph_StoreError(t *testing.T) {
	svc := NewIndexerService(nil, nil, nil, nil, &mockStore{allFilesErr: errors.New("all files failed")})
	_, err := svc.BuildReferenceGraph()
	if err == nil {
		t.Fatal("expected AllFiles error")
	}
}

func TestIndexerService_TopicsForFile_NotFound(t *testing.T) {
	svc := NewIndexerService(nil, nil, nil, nil, &mockStore{getFileFound: false})
	topics, err := svc.TopicsForFile("/root/nonexistent.md")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if topics != nil {
		t.Fatalf("expected nil topics for a file that was never indexed, got %v", topics)
	}
}

func TestIndexerService_TopicsForFile_StoreError(t *testing.T) {
	svc := NewIndexerService(nil, nil, nil, nil, &mockStore{getFileErr: errors.New("get file failed")})
	_, err := svc.TopicsForFile("/root/a.md")
	if err == nil {
		t.Fatal("expected GetFile error")
	}
}

func TestIndexerService_FilesRelatedByTopic_FilesByTopicError(t *testing.T) {
	store := &mockStore{
		getFile:               &domain.FileRecord{Path: "/root/a.md", Topics: []string{"go"}},
		getFileFound:          true,
		filesByTopicErrOnCall: 1,
	}
	svc := NewIndexerService(nil, nil, nil, nil, store)
	_, err := svc.FilesRelatedByTopic("/root/a.md")
	if err == nil {
		t.Fatal("expected FilesByTopic error")
	}
}
