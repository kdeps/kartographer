package infrastructure

import (
	"path/filepath"
	"strings"

	"github.com/kdeps/kartographer/graph/index/domain"
)

// topicExtensions are the extensions CompositeTopicExtractor treats as
// "bare" YAML/JSON documents -- the whole file is unmarshaled as if it were
// a frontmatter block's inner content, no leading "---" markers required.
//
//nolint:gochecknoglobals // static lookup table
var topicExtensions = map[string]bool{".yaml": true, ".yml": true, ".json": true}

// CompositeReferenceExtractor dispatches to a per-format ReferenceExtractor
// by fromPath's extension: markdown/wikilinks for .md/.markdown/.txt,
// href/src for HTML, import/include resolution for recognized source code
// extensions. Any other extension (including bare .yaml/.yml/.json, which
// only get topic extraction) yields no references.
type CompositeReferenceExtractor struct {
	markdown domain.ReferenceExtractor
	html     domain.ReferenceExtractor
	code     *SourceCodeReferenceExtractor
}

// NewCompositeReferenceExtractor builds the dispatching extractor. root is
// the indexed root every sub-extractor resolves relative paths against and
// enforces containment within.
func NewCompositeReferenceExtractor(root string) domain.ReferenceExtractor {
	return &CompositeReferenceExtractor{
		markdown: NewLinkReferenceExtractor(root),
		html:     NewHTMLReferenceExtractor(root),
		code:     &SourceCodeReferenceExtractor{Root: root},
	}
}

// SetKnownFiles implements domain.FileSetAware, forwarding to the source
// code sub-extractor (the only one that needs the full file set, for
// resolving non-relative imports).
func (e *CompositeReferenceExtractor) SetKnownFiles(paths []string) {
	e.code.SetKnownFiles(paths)
}

func (e *CompositeReferenceExtractor) Extract(content, fromPath string) []string {
	ext := strings.ToLower(filepath.Ext(fromPath))
	switch {
	case ext == ".md" || ext == ".markdown" || ext == ".txt":
		return e.markdown.Extract(content, fromPath)
	case ext == ".html" || ext == ".htm":
		return e.html.Extract(content, fromPath)
	case IsSourceCodeExtension(ext):
		return e.code.Extract(content, fromPath)
	default:
		return nil
	}
}

// CompositeTopicExtractor extracts topics from a leading YAML frontmatter
// block (markdown behavior, unchanged) and, when that finds nothing and
// fromPath is a bare .yaml/.yml/.json file, from a top-level "topics:"/
// "tags:" key in the whole document -- JSON is valid YAML, so the same
// unmarshal handles both.
type CompositeTopicExtractor struct {
	frontmatter domain.TopicExtractor
}

func NewCompositeTopicExtractor() domain.TopicExtractor {
	return &CompositeTopicExtractor{frontmatter: NewFrontmatterTopicExtractor()}
}

func (e *CompositeTopicExtractor) Extract(content, fromPath string) []string {
	if topics := e.frontmatter.Extract(content, fromPath); len(topics) > 0 {
		return topics
	}
	if !topicExtensions[strings.ToLower(filepath.Ext(fromPath))] {
		return nil
	}
	return topicsFromYAML(content)
}
