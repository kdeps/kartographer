package infrastructure

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/afero"

	"github.com/kdeps/kartographer/graph/index/domain"
)

type AferoFileWalker struct {
	Fs afero.Fs
}

func NewAferoFileWalker(fs afero.Fs) domain.FileWalker {
	return &AferoFileWalker{Fs: fs}
}

func (w *AferoFileWalker) Walk(root string, extensions []string) ([]string, error) {
	allowed := make(map[string]bool, len(extensions))
	for _, ext := range extensions {
		allowed[strings.ToLower(ext)] = true
	}

	var paths []string
	err := afero.Walk(w.Fs, root, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil || info == nil || info.IsDir() {
			return nil //nolint:nilerr // skip unreadable entries and directories
		}
		if allowed[strings.ToLower(filepath.Ext(path))] {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return paths, nil
}
