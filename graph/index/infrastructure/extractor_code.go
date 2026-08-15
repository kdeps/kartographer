package infrastructure

import (
	"path/filepath"
	"regexp"
	"strings"

	"github.com/kdeps/kartographer/graph/index/domain"
)

// Per-language import/include patterns. Heuristic regex, not full parsers:
// each captures the raw import target string for further resolution.
var (
	goImportPattern       = regexp.MustCompile(`import\s+"([^"]+)"`)
	pyRelativeFromPattern = regexp.MustCompile(`(?m)^\s*from\s+(\.+[\w.]*)\s+import`)
	pyImportPattern       = regexp.MustCompile(`(?m)^\s*import\s+([\w.]+)`)
	pyFromPattern         = regexp.MustCompile(`(?m)^\s*from\s+([\w][\w.]*)\s+import`)
	rustModPattern        = regexp.MustCompile(`(?m)^\s*mod\s+(\w+)\s*;`)
	rustUsePattern        = regexp.MustCompile(`(?m)^\s*use\s+([\w:]+)`)
	jsImportPattern       = regexp.MustCompile(`import\s+(?:[^'"]*?\s+from\s+)?['"]([^'"]+)['"]`)
	jsRequirePattern      = regexp.MustCompile(`require\(\s*['"]([^'"]+)['"]\s*\)`)
	cIncludeLocalPattern  = regexp.MustCompile(`#include\s*"([^"]+)"`)
	rbRequireRelPattern   = regexp.MustCompile(`require_relative\s+['"]([^'"]+)['"]`)
	rbRequirePattern      = regexp.MustCompile(`require\s+['"]([^'"]+)['"]`)
	javaImportPattern     = regexp.MustCompile(`(?m)^\s*import\s+([\w.]+)\s*;`)
)

// codeExtensions maps each recognized source extension to a language group,
// mirroring the groups kdeps's codeIntelligence LSP dispatch uses (go,
// python, rust, typescript/javascript, c/cpp, ruby, java).
//
//nolint:gochecknoglobals // static lookup table
var codeExtensions = map[string]bool{
	".go": true, ".py": true, ".rs": true,
	".ts": true, ".tsx": true, ".js": true, ".jsx": true,
	".c": true, ".h": true, ".cpp": true, ".hpp": true, ".cc": true,
	".rb": true, ".java": true,
}

// IsSourceCodeExtension reports whether ext (as returned by filepath.Ext,
// including the leading dot) is one SourceCodeReferenceExtractor handles.
func IsSourceCodeExtension(ext string) bool {
	return codeExtensions[strings.ToLower(ext)]
}

// SourceCodeReferenceExtractor extracts import/include targets from source
// files across the languages kdeps's codeIntelligence LSP dispatch supports.
// Relative imports resolve like a markdown link (relative to fromPath,
// dropped if outside the indexed root). Non-relative imports (module paths,
// dotted package names) resolve by matching against the file set the
// extractor learns via SetKnownFiles, and are dropped -- not guessed -- when
// no unambiguous match exists (e.g. an external/stdlib import).
type SourceCodeReferenceExtractor struct {
	Root string

	built     bool
	files     map[string]bool
	baseIndex map[string][]string // base name (file without ext, or dir name) -> full paths
}

func NewSourceCodeReferenceExtractor(root string) domain.ReferenceExtractor {
	return &SourceCodeReferenceExtractor{Root: root}
}

// SetKnownFiles implements domain.FileSetAware.
func (e *SourceCodeReferenceExtractor) SetKnownFiles(paths []string) {
	e.files = make(map[string]bool, len(paths))
	e.baseIndex = make(map[string][]string)
	dirsSeen := make(map[string]bool)
	for _, p := range paths {
		e.files[p] = true
		nameNoExt := strings.TrimSuffix(filepath.Base(p), filepath.Ext(p))
		e.baseIndex[nameNoExt] = append(e.baseIndex[nameNoExt], p)

		dir := filepath.Dir(p)
		if !dirsSeen[dir] {
			dirsSeen[dir] = true
			e.baseIndex[filepath.Base(dir)] = append(e.baseIndex[filepath.Base(dir)], dir)
		}
	}
	e.built = true
}

func (e *SourceCodeReferenceExtractor) Extract(content, fromPath string) []string {
	if !e.built {
		// No known-file set yet (e.g. called outside IndexFolder): only
		// relative resolution is possible.
		e.files = map[string]bool{}
		e.baseIndex = map[string][]string{}
	}

	var refs []string
	switch strings.ToLower(filepath.Ext(fromPath)) {
	case ".go":
		refs = e.extractGo(content)
	case ".py":
		refs = e.extractPython(content, fromPath)
	case ".rs":
		refs = e.extractRust(content, fromPath)
	case ".ts", ".tsx", ".js", ".jsx":
		refs = e.extractJSLike(content, fromPath)
	case ".c", ".h", ".cpp", ".hpp", ".cc":
		refs = e.extractC(content, fromPath)
	case ".rb":
		refs = e.extractRuby(content, fromPath)
	case ".java":
		refs = e.extractJava(content)
	}
	return dedupeNonEmpty(refs)
}

func dedupeNonEmpty(refs []string) []string {
	if len(refs) == 0 {
		return nil
	}
	seen := make(map[string]bool, len(refs))
	var out []string
	for _, r := range refs {
		if r == "" || seen[r] {
			continue
		}
		seen[r] = true
		out = append(out, r)
	}
	return out
}

// resolveDotted resolves a dotted/slashed module path (e.g. "foo.bar",
// "foo::bar", "foo/bar") against the known file set: first as an exact file
// (segments joined + each candidate extension), then as a package directory
// (segments joined + each indexName inside it), and finally -- only when
// unambiguous -- by matching just the last segment's base name anywhere
// under root.
func (e *SourceCodeReferenceExtractor) resolveDotted(segments []string, exts, indexNames []string) (string, bool) {
	if len(segments) == 0 {
		return "", false
	}
	base := filepath.Join(e.Root, filepath.Join(segments...))
	for _, ext := range exts {
		if p := base + ext; e.files[p] {
			return p, true
		}
	}
	for _, idx := range indexNames {
		if p := filepath.Join(base, idx); e.files[p] {
			return p, true
		}
	}
	last := segments[len(segments)-1]
	if matches := e.baseIndex[last]; len(matches) == 1 {
		return matches[0], true
	}
	return "", false
}

func (e *SourceCodeReferenceExtractor) extractGo(content string) []string {
	var refs []string
	for _, m := range goImportPattern.FindAllStringSubmatch(content, -1) {
		segments := strings.Split(m[1], "/")
		if p, ok := e.resolveDotted(segments, nil, nil); ok {
			refs = append(refs, p)
		}
	}
	return refs
}

func (e *SourceCodeReferenceExtractor) extractPython(content, fromPath string) []string {
	var refs []string
	for _, m := range pyRelativeFromPattern.FindAllStringSubmatch(content, -1) {
		leadingDots := len(m[1]) - len(strings.TrimLeft(m[1], "."))
		rest := m[1][leadingDots:]
		if rest == "" {
			// "from . import x" / "from .. import x": imports the package
			// itself, not a clearly identified sibling file -- skip.
			continue
		}
		target := strings.Repeat("../", leadingDots-1) + strings.ReplaceAll(rest, ".", "/") + ".py"
		if p, ok := resolveRelative(e.Root, fromPath, target); ok {
			refs = append(refs, p)
		}
	}
	dottedPatterns := []*regexp.Regexp{pyImportPattern, pyFromPattern}
	for _, pat := range dottedPatterns {
		for _, m := range pat.FindAllStringSubmatch(content, -1) {
			segments := strings.Split(m[1], ".")
			if p, ok := e.resolveDotted(segments, []string{".py"}, []string{"__init__.py"}); ok {
				refs = append(refs, p)
			}
		}
	}
	return refs
}

func (e *SourceCodeReferenceExtractor) extractRust(content, fromPath string) []string {
	var refs []string
	for _, m := range rustModPattern.FindAllStringSubmatch(content, -1) {
		name := m[1]
		// Two mutually-exclusive candidate layouts ("foo.rs" vs
		// "foo/mod.rs"): unlike a single-candidate relative link, checking
		// e.files here picks whichever one actually exists.
		if p, ok := resolveRelative(e.Root, fromPath, name+".rs"); ok && e.files[p] {
			refs = append(refs, p)
		} else if p, ok := resolveRelative(e.Root, fromPath, filepath.Join(name, "mod.rs")); ok && e.files[p] {
			refs = append(refs, p)
		}
	}
	for _, m := range rustUsePattern.FindAllStringSubmatch(content, -1) {
		segments := strings.Split(m[1], "::")
		segments = trimLeadingRustKeywords(segments)
		if p, ok := e.resolveDotted(segments, []string{".rs"}, []string{"mod.rs"}); ok {
			refs = append(refs, p)
		}
	}
	return refs
}

func trimLeadingRustKeywords(segments []string) []string {
	for len(segments) > 0 && (segments[0] == "crate" || segments[0] == "self" || segments[0] == "super") {
		segments = segments[1:]
	}
	return segments
}

func (e *SourceCodeReferenceExtractor) extractJSLike(content, fromPath string) []string {
	var refs []string
	patterns := []*regexp.Regexp{jsImportPattern, jsRequirePattern}
	exts := []string{".ts", ".tsx", ".js", ".jsx"}
	for _, pat := range patterns {
		for _, m := range pat.FindAllStringSubmatch(content, -1) {
			target := m[1]
			if strings.HasPrefix(target, ".") {
				if p, ok := e.resolveJSRelative(fromPath, target, exts); ok {
					refs = append(refs, p)
				}
				continue
			}
			segments := strings.Split(target, "/")
			var indexNames []string
			for _, ext := range exts {
				indexNames = append(indexNames, "index"+ext)
			}
			if p, ok := e.resolveDotted(segments, exts, indexNames); ok {
				refs = append(refs, p)
			}
		}
	}
	return refs
}

// resolveJSRelative resolves a relative specifier that may omit its
// extension (e.g. "./util" -> "./util.ts"), trying each candidate extension
// in turn before giving up.
func (e *SourceCodeReferenceExtractor) resolveJSRelative(fromPath, target string, exts []string) (string, bool) {
	if p, ok := resolveRelative(e.Root, fromPath, target); ok && e.files[p] {
		return p, true
	}
	for _, ext := range exts {
		if p, ok := resolveRelative(e.Root, fromPath, target+ext); ok && e.files[p] {
			return p, true
		}
	}
	return "", false
}

func (e *SourceCodeReferenceExtractor) extractC(content, fromPath string) []string {
	var refs []string
	for _, m := range cIncludeLocalPattern.FindAllStringSubmatch(content, -1) {
		// <...> system includes are intentionally not matched -- they're
		// never local files.
		if p, ok := resolveRelative(e.Root, fromPath, m[1]); ok {
			refs = append(refs, p)
		}
	}
	return refs
}

func (e *SourceCodeReferenceExtractor) extractRuby(content, fromPath string) []string {
	var refs []string
	for _, m := range rbRequireRelPattern.FindAllStringSubmatch(content, -1) {
		target := m[1]
		if !strings.HasSuffix(target, ".rb") {
			target += ".rb"
		}
		if p, ok := resolveRelative(e.Root, fromPath, target); ok {
			refs = append(refs, p)
		}
	}
	for _, m := range rbRequirePattern.FindAllStringSubmatch(content, -1) {
		segments := strings.Split(m[1], "/")
		if p, ok := e.resolveDotted(segments, []string{".rb"}, nil); ok {
			refs = append(refs, p)
		}
	}
	return refs
}

func (e *SourceCodeReferenceExtractor) extractJava(content string) []string {
	var refs []string
	for _, m := range javaImportPattern.FindAllStringSubmatch(content, -1) {
		segments := strings.Split(m[1], ".")
		if p, ok := e.resolveDotted(segments, []string{".java"}, nil); ok {
			refs = append(refs, p)
		}
	}
	return refs
}
