package graph

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNearest(t *testing.T) {
	edges := map[string][]string{
		"a": {"b", "c"},
		"b": {"d", "gone"},
		"c": {"d", "a"},
		"d": nil,
	}
	next := func(n string) ([]string, bool) {
		s, ok := edges[n]
		return s, ok
	}
	assert.Equal(t, []string{"a", "b", "c", "d"}, Nearest("a", 10, next), "closest first; missing nodes skipped; cycles stop")
	assert.Equal(t, []string{"a", "b"}, Nearest("a", 2, next))
	assert.Empty(t, Nearest("a", 0, next))
	assert.Empty(t, Nearest("gone", 5, next), "a missing start returns nothing")
}
