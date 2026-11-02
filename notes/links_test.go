package notes

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWikiLinkTargetsDropAliasesAndAnchors(t *testing.T) {
	got := WikiLinkTargets("see [[ Alpha ]], [[beta|the B]], [[gamma#Heading]] and [[delta^block]].")
	assert.Equal(t, []string{"Alpha", "beta", "gamma", "delta"}, got)
}

func TestMarkdownNoteLinksSkipURLs(t *testing.T) {
	got := MarkdownNoteLinks("[a](notes/a.md) [b](https://x.test/b.md) [c](c.md#part) [d](d.txt)")
	assert.Equal(t, []string{"notes/a.md", "c.md"}, got)
}

func TestStripInlineCodeRemovesSpans(t *testing.T) {
	assert.Equal(t, "see  and [[x]]", StripInlineCode("see `[[code]]` and [[x]]"))
}
