package search

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
)

func TestSnippet(t *testing.T) {
	c := "aaaaaaaaaa MATCH bbbbbbbbbb"
	i := strings.Index(c, "MATCH")
	assert.Equal(t, "...aa MATCH bb...", Snippet(c, i, i+5, 3))
	assert.Equal(t, c, Snippet(c, i, i+5, 100), "context past both ends: no marks")
	assert.Equal(t, "aaaaaa...", Snippet(c, -1, -1, 3), "no match: the start, 2*context long")
	assert.Equal(t, "ab", Snippet("ab", -1, -1, 3))
	assert.Equal(t, "aaaaaa...", Snippet(c, len(c)+5, len(c)+6, 3), "a match past the end reads as none")

	u := "ééééé X ééééé"
	j := strings.Index(u, "X")
	s := Snippet(u, j, j+1, 2)
	assert.True(t, utf8.ValidString(s), s)
	assert.Contains(t, s, "X")
	assert.True(t, utf8.ValidString(Snippet(u, -1, -1, 1)))
}
