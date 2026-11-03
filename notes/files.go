package notes

import (
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
)

// FileOptions say which files under a root are notes. The caller owns them.
type FileOptions struct {
	Ext     string // the note file extension, such as ".md", matched without case
	MaxSize int64  // larger files are left out; 0 keeps every size
	// Skip leaves out a file the caller does not count as a note (such as a
	// generated report); rel is the slash path under the root. Nil skips none.
	Skip func(rel, abs string) bool
}

// File is one note file under a root.
type File struct {
	Rel     string // slash path under the root
	Abs     string
	Size    int64
	ModTime int64 // Unix nanoseconds
}

// Files lists the note files under root, sorted by path. Hidden files and
// folders (a name starting with a dot) are left out, and so is a file whose
// metadata cannot be read, so one bad file does not fail the listing.
func Files(root string, o FileOptions) ([]File, error) {
	var out []File
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if strings.HasPrefix(d.Name(), ".") && p != root {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() || !strings.EqualFold(filepath.Ext(p), o.Ext) {
			return nil
		}
		info, infoErr := d.Info()
		if infoErr != nil || (o.MaxSize > 0 && info.Size() > o.MaxSize) {
			return nil //nolint:nilerr // an unreadable or oversized file is left out
		}
		rel, relErr := filepath.Rel(root, p)
		if relErr != nil {
			return relErr
		}
		rel = filepath.ToSlash(rel)
		if o.Skip != nil && o.Skip(rel, p) {
			return nil
		}
		out = append(out, File{Rel: rel, Abs: p, Size: info.Size(), ModTime: info.ModTime().UnixNano()})
		return nil
	})
	sort.Slice(out, func(i, j int) bool { return out[i].Rel < out[j].Rel })
	return out, err
}

// ManifestEntry is a note file's size and modification time.
type ManifestEntry struct {
	Size    int64 `json:"size"`
	ModTime int64 `json:"modTime"`
}

// Manifest records the state of every note file, by slash path, so a caller
// can tell whether any was added, changed, renamed or removed since it was taken.
type Manifest map[string]ManifestEntry

// ManifestOf is the manifest of files.
func ManifestOf(files []File) Manifest {
	m := make(Manifest, len(files))
	for _, f := range files {
		m[f.Rel] = ManifestEntry{Size: f.Size, ModTime: f.ModTime}
	}
	return m
}

// Equal reports whether two manifests record the same files in the same state.
func (m Manifest) Equal(other Manifest) bool {
	if len(m) != len(other) {
		return false
	}
	for k, v := range m {
		if o, ok := other[k]; !ok || o != v {
			return false
		}
	}
	return true
}
