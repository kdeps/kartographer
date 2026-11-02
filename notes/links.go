package notes

import (
	"regexp"
	"strings"
)

//nolint:gochecknoglobals // compiled once; the patterns are the link syntax
var (
	wikiLinkRe   = regexp.MustCompile(`\[\[([^\]|#^]+)(?:[#^][^\]|]*)?(?:\|[^\]]*)?\]\]`)
	mdLinkRe     = regexp.MustCompile(`\[[^\]]*\]\(([^)\s]+?\.md)(?:#[^)]*)?\)`)
	inlineCodeRe = regexp.MustCompile("`[^`]*`")
)

// WikiLinkTargets returns the target of each [[wiki link]] in text, trimmed of
// spaces. A heading or block reference and an alias are not part of the target.
func WikiLinkTargets(text string) []string {
	var out []string
	for _, m := range wikiLinkRe.FindAllStringSubmatch(text, -1) {
		out = append(out, strings.TrimSpace(m[1]))
	}
	return out
}

// MarkdownNoteLinks returns the target of each Markdown link to a .md file in
// text. A link to a URL is not a note, and is left out.
func MarkdownNoteLinks(text string) []string {
	var out []string
	for _, m := range mdLinkRe.FindAllStringSubmatch(text, -1) {
		if !strings.Contains(m[1], "://") {
			out = append(out, m[1])
		}
	}
	return out
}

// StripInlineCode removes `code spans` from a line, so links inside them are
// not read.
func StripInlineCode(line string) string {
	return inlineCodeRe.ReplaceAllString(line, "")
}
