package search

import (
	"path/filepath"
	"slices"
	"strings"
)

// FilterOptions are the rules of which files an index reads. The caller owns them.
type FilterOptions struct {
	SkipHidden     bool     // names starting with a dot are never read
	SkipDirs       []string // folder names never walked
	Extensions     []string // file extensions read; a file without one always is
	SecretNames    []string // file names that hold keys or credentials
	SecretPrefixes []string // file name prefixes that do (such as ".env.")
	SecretExts     []string // extensions that do (such as ".pem")
	SecretDirs     []string // folders whose files all do (such as ".ssh")
}

// Filter decides which files an index reads, and which are secrets that are
// never read or returned. Folder names and extensions match exactly; secret
// names, prefixes, extensions and folders match without case.
type Filter struct {
	o                                   FilterOptions
	skipDirs, exts, names, sExts, sDirs map[string]bool
}

// NewFilter returns a filter that follows o.
func NewFilter(o FilterOptions) *Filter {
	set := func(ws []string, fold bool) map[string]bool {
		m := make(map[string]bool, len(ws))
		for _, w := range ws {
			if fold {
				w = strings.ToLower(w)
			}
			m[w] = true
		}
		return m
	}
	return &Filter{o: o, skipDirs: set(o.SkipDirs, false), exts: set(o.Extensions, false),
		names: set(o.SecretNames, true), sExts: set(o.SecretExts, true), sDirs: set(o.SecretDirs, true)}
}

// SkipEntry reports a walk entry that is never read: an empty name, a hidden
// one when hidden names are skipped, or a skipped folder name.
func (f *Filter) SkipEntry(name string) bool {
	if name == "" {
		return true
	}
	if f.o.SkipHidden && name[0] == '.' {
		return true
	}
	return f.skipDirs[name]
}

// Indexable reports a file whose text an index reads: not a secret, and with
// one of the extensions (or none).
func (f *Filter) Indexable(name string) bool {
	if f.Secret(name) {
		return false
	}
	ext := filepath.Ext(name)
	return ext == "" || f.exts[ext]
}

// Secret reports a file that holds keys or credentials, by its name, a name
// prefix, its extension, or a folder on its path.
func (f *Filter) Secret(path string) bool {
	name := strings.ToLower(filepath.Base(path))
	if f.names[name] || f.sExts[filepath.Ext(name)] ||
		slices.ContainsFunc(f.o.SecretPrefixes, func(p string) bool { return strings.HasPrefix(name, strings.ToLower(p)) }) {
		return true
	}
	for _, part := range strings.Split(filepath.ToSlash(filepath.Dir(path)), "/") {
		if f.sDirs[strings.ToLower(part)] {
			return true
		}
	}
	return false
}
