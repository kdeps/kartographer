package infrastructure

import (
	"regexp"

	"github.com/kdeps/kartographer/graph/index/domain"
)

// htmlAttrPattern matches href="..." / src="..." attribute values (single
// or double quoted) on any HTML element -- <a href>, <link href>,
// <script src>, <img src>, etc.
var htmlAttrPattern = regexp.MustCompile(`(?:href|src)\s*=\s*["']([^"']+)["']`)

// HTMLReferenceExtractor extracts href/src attribute values from HTML
// content, resolving them the same way LinkReferenceExtractor resolves
// markdown links: relative to the containing file, dropped if external or
// outside the indexed root.
type HTMLReferenceExtractor struct {
	Root string
}

func NewHTMLReferenceExtractor(root string) domain.ReferenceExtractor {
	return &HTMLReferenceExtractor{Root: root}
}

func (e *HTMLReferenceExtractor) Extract(content, fromPath string) []string {
	seen := make(map[string]bool)
	var refs []string
	for _, m := range htmlAttrPattern.FindAllStringSubmatch(content, -1) {
		target := m[1]
		if isExternalURL(target) {
			continue
		}
		resolved, ok := resolveRelative(e.Root, fromPath, target)
		if !ok || seen[resolved] {
			continue
		}
		seen[resolved] = true
		refs = append(refs, resolved)
	}
	return refs
}
