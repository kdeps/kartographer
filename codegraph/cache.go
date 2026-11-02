package codegraph

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// cacheVersion changes when the saved format or the analysis changes.
const cacheVersion = 5

// restoreInferred rebuilds the inferred-edge set saved by saveSnapshot.
func restoreInferred(in map[string][]string) map[string]map[string]bool {
	out := map[string]map[string]bool{}
	for from, tos := range in {
		out[from] = map[string]bool{}
		for _, to := range tos {
			out[from][to] = true
		}
	}
	return out
}

// snapshot is an index saved to disk.
type snapshot struct {
	Version     int                    `json:"version"`
	Fingerprint string                 `json:"fingerprint"`
	Module      string                 `json:"module"`
	Symbols     map[string]Symbol      `json:"symbols"`
	Missing     []string               `json:"missing,omitempty"`
	Deps        map[string][]string    `json:"deps"`
	Refs        map[string][]Reference `json:"refs,omitempty"`
	Inferred    map[string][]string    `json:"inferred,omitempty"`
}

// BuildCached returns the index of dir, reusing the copy saved in cacheDir
// when no file under dir has changed since it was saved. Otherwise it builds
// the index and saves it there.
func BuildCached(dir, cacheDir string) (*Index, error) {
	root, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	fp, err := Fingerprint(root)
	if err != nil {
		return nil, err
	}
	file := filepath.Join(cacheDir, cacheName(root))
	if ix, ok := loadSnapshot(file, fp, root); ok {
		return ix, nil
	}
	ix, err := Build(root)
	if err != nil {
		return nil, err
	}
	return ix, saveSnapshot(file, fp, ix)
}

// cacheName is the cache file for a module root.
func cacheName(root string) string {
	sum := sha256.Sum256([]byte(root))
	return hex.EncodeToString(sum[:8]) + ".json"
}

// Fingerprint hashes the path, size and modification time of every file under
// root, so any edit, addition or removal changes it.
func Fingerprint(root string) (string, error) {
	h := sha256.New()
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if p != root && (strings.HasPrefix(d.Name(), ".") || skipDirs[d.Name()]) {
				return filepath.SkipDir
			}
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		fmt.Fprintf(h, "%s\x00%d\x00%d\n", p, info.Size(), info.ModTime().UnixNano())
		return nil
	})
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// loadSnapshot restores the index from file when it matches fp and root.
func loadSnapshot(file, fp, root string) (*Index, bool) {
	raw, err := os.ReadFile(file)
	if err != nil {
		return nil, false
	}
	var s snapshot
	if json.Unmarshal(raw, &s) != nil || s.Version != cacheVersion || s.Fingerprint != fp {
		return nil, false
	}
	ix := &Index{Root: root, Module: s.Module, Symbols: s.Symbols, Missing: s.Missing, refs: s.Refs,
		inferred: restoreInferred(s.Inferred)}
	ix.setDeps(s.Deps)
	if err := ix.attachNotes(); err != nil {
		return nil, false
	}
	return ix, true
}

// saveSnapshot writes the index to file, creating its directory.
func saveSnapshot(file, fp string, ix *Index) error {
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		return err
	}
	deps := map[string][]string{}
	for name := range ix.Symbols {
		deps[name] = ix.deps.NodeDependencies[name]
	}
	inferred := map[string][]string{}
	for from, tos := range ix.inferred {
		for to := range tos {
			inferred[from] = append(inferred[from], to)
		}
	}
	raw, err := json.Marshal(snapshot{
		Version: cacheVersion, Fingerprint: fp, Module: ix.Module,
		Symbols: ix.Symbols, Missing: ix.Missing, Deps: deps, Refs: ix.refs,
		Inferred: inferred,
	})
	if err != nil {
		return err
	}
	return os.WriteFile(file, raw, 0o600)
}

// CacheDir is the default directory for saved indexes.
func CacheDir() (string, error) {
	dir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "kartographer", "index"), nil
}
