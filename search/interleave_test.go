package search

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInterleave(t *testing.T) {
	id := func(s string) string { return s }
	lists := [][]string{{"a1", "a2", "a3"}, {"b1", "a1"}}
	assert.Equal(t, []string{"a1", "b1", "a2", "a3"}, Interleave(lists, 4, id), "turns; a key seen once")
	assert.Equal(t, []string{"a1", "b1"}, Interleave(lists, 2, id))
	assert.Equal(t, []string{"a1", "b1", "a2", "a3"}, Interleave(lists, 10, id), "stops when every list is spent")
	assert.Empty(t, Interleave[string](nil, 3, id))
	assert.Empty(t, Interleave(lists, 0, id))
}
