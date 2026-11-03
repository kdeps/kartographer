package search

import "unicode/utf8"

// Snippet is content around the match [start, end): up to context bytes on
// each side, widened to whole characters, with "..." where content goes on.
// With no usable match (start < 0, past the end, or end before start) it is
// the first 2*context bytes of content.
func Snippet(content string, start, end, context int) string {
	if start < 0 || start >= len(content) || end < start {
		p := cut(content, 2*context) //nolint:mnd // both sides of a match
		if len(p) < len(content) {
			return p + "..."
		}
		return p
	}
	from, to := max(0, start-context), min(len(content), end+context)
	for from > 0 && !utf8.RuneStart(content[from]) {
		from--
	}
	for to < len(content) && !utf8.RuneStart(content[to]) {
		to++
	}
	s := content[from:to]
	if from > 0 {
		s = "..." + s
	}
	if to < len(content) {
		s += "..."
	}
	return s
}
