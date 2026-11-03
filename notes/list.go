package notes

import (
	"regexp"
	"strings"
)

// listMarkerRe matches a list item's marker and checkbox: "- ", "2. ", "- [x] ".
var listMarkerRe = regexp.MustCompile(`^\s*(?:[-*+]|\d+[.)])?\s*(?:\[[ xX]\])?\s*`) //nolint:gochecknoglobals // fixed pattern

// ListItem is line without its list marker and checkbox, trimmed.
func ListItem(line string) string {
	return strings.TrimSpace(listMarkerRe.ReplaceAllString(line, ""))
}

// ListItems is the non-empty lines of text as list items (see ListItem).
func ListItems(text string) []string {
	var out []string
	for _, l := range strings.Split(text, "\n") {
		if l = ListItem(l); l != "" {
			out = append(out, l)
		}
	}
	return out
}
