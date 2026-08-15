package infrastructure

import (
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/kdeps/kartographer/graph/index/domain"
)

var (
	markdownLinkPattern = regexp.MustCompile(`\[[^\]]*\]\(([^)]+)\)`)
	wikilinkPattern     = regexp.MustCompile(`\[\[([^\]|]+)(?:\|[^\]]*)?\]\]`)
	frontmatterPattern  = regexp.MustCompile(`(?s)\A---\r?\n(.*?)\r?\n---\r?\n`)
)

type frontmatter struct {
	Topics []string `yaml:"topics"`
	Tags   []string `yaml:"tags"`
}

// LinkReferenceExtractor extracts markdown links and wikilinks, resolving
// them relative to the containing file and dropping anything that resolves
// outside the indexed root or points at an external URL.
type LinkReferenceExtractor struct {
	Root string
}

func NewLinkReferenceExtractor(root string) domain.ReferenceExtractor {
	return &LinkReferenceExtractor{Root: root}
}

func (e *LinkReferenceExtractor) Extract(content, fromPath string) []string {
	var raw []string
	for _, m := range markdownLinkPattern.FindAllStringSubmatch(content, -1) {
		raw = append(raw, m[1])
	}
	for _, m := range wikilinkPattern.FindAllStringSubmatch(content, -1) {
		raw = append(raw, m[1])
	}

	seen := make(map[string]bool)
	var refs []string
	for _, target := range raw {
		resolved, ok := e.resolve(fromPath, target)
		if !ok || seen[resolved] {
			continue
		}
		seen[resolved] = true
		refs = append(refs, resolved)
	}
	return refs
}

func (e *LinkReferenceExtractor) resolve(fromPath, target string) (string, bool) {
	if isExternalURL(target) {
		return "", false
	}
	return resolveRelative(e.Root, fromPath, target)
}

// resolveRelative resolves target relative to fromPath's directory and
// returns it only if it stays inside root -- the containment rule shared by
// every extractor that resolves relative references (markdown links, HTML
// href/src, relative source-code imports).
func resolveRelative(root, fromPath, target string) (string, bool) {
	target = strings.TrimSpace(target)
	if target == "" {
		return "", false
	}

	dir := filepath.Dir(fromPath)
	resolved := filepath.Clean(filepath.Join(dir, target))

	rel, err := filepath.Rel(root, resolved)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", false
	}
	return resolved, true
}

func isExternalURL(target string) bool {
	lower := strings.ToLower(target)
	return strings.HasPrefix(lower, "http://") ||
		strings.HasPrefix(lower, "https://") ||
		strings.HasPrefix(lower, "mailto:") ||
		strings.HasPrefix(lower, "//")
}

// FrontmatterTopicExtractor extracts a `topics:` or `tags:` list from a
// leading YAML frontmatter block.
type FrontmatterTopicExtractor struct{}

func NewFrontmatterTopicExtractor() domain.TopicExtractor {
	return &FrontmatterTopicExtractor{}
}

func (e *FrontmatterTopicExtractor) Extract(content, _ string) []string {
	m := frontmatterPattern.FindStringSubmatch(content)
	if m == nil {
		return nil
	}
	return topicsFromYAML(m[1])
}

// topicsFromYAML unmarshals src (a YAML document, or a frontmatter block's
// inner content) into topics/tags, deduped and order-preserved.
func topicsFromYAML(src string) []string {
	var fm frontmatter
	if err := yaml.Unmarshal([]byte(src), &fm); err != nil {
		return nil
	}

	seen := make(map[string]bool)
	var topics []string
	for _, t := range append(fm.Topics, fm.Tags...) {
		t = strings.TrimSpace(t)
		if t == "" || seen[t] {
			continue
		}
		seen[t] = true
		topics = append(topics, t)
	}
	return topics
}
