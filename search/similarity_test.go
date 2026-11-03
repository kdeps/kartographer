package search

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func set(ws ...string) map[string]bool {
	m := map[string]bool{}
	for _, w := range ws {
		m[w] = true
	}
	return m
}

func TestJaccard(t *testing.T) {
	assert.InDelta(t, 1.0/3, Jaccard(set("a", "b"), set("b", "c")), 1e-9)
	assert.InDelta(t, 1.0, Jaccard(set("a"), set("a")), 1e-9)
	assert.Zero(t, Jaccard(nil, set("a")))
	assert.Zero(t, Jaccard(set("a"), nil))
	assert.Zero(t, Jaccard(set("a"), set("b")))
}

func TestCosine(t *testing.T) {
	assert.InDelta(t, 1.0, Cosine([]float32{1, 2}, []float32{2, 4}), 1e-9)
	assert.InDelta(t, 0.0, Cosine([]float32{1, 0}, []float32{0, 1}), 1e-9)
	assert.Zero(t, Cosine([]float32{0, 0}, []float32{1, 1}))
	assert.Zero(t, Cosine([]float32{1}, []float32{1, 2}))
	assert.Zero(t, Cosine(nil, nil))
}

func TestTopK(t *testing.T) {
	s := []float64{0.1, 0.9, 0.5, 0.9, 0.2}
	assert.Equal(t, []int{1, 3}, TopK(s, 2), "best two, in index order")
	assert.Equal(t, []int{1, 2, 3}, TopK(s, 3))
	assert.Equal(t, []int{0, 1, 2, 3, 4}, TopK(s, 0))
	assert.Equal(t, []int{0, 1, 2, 3, 4}, TopK(s, 9))
	assert.Equal(t, []int{0}, TopK([]float64{1, 1}, 1), "a tie keeps the earlier index")
	assert.Empty(t, TopK(nil, 2))
}
