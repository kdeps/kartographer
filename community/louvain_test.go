package community

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func clique(nodes ...int) []Edge {
	var out []Edge
	for i := range nodes {
		for j := i + 1; j < len(nodes); j++ {
			out = append(out, Edge{From: nodes[i], To: nodes[j], Weight: 1})
		}
	}
	return out
}

func TestTwoCliquesJoinedByOneEdgeSplit(t *testing.T) {
	edges := append(clique(0, 1, 2), clique(3, 4, 5)...)
	edges = append(edges, Edge{From: 2, To: 3, Weight: 1})
	assign, q := Detect(6, edges)
	assert.Equal(t, []int{0, 0, 0, 1, 1, 1}, assign)
	assert.Greater(t, q, 0.3)
}

func TestLargestCommunityIsZero(t *testing.T) {
	edges := append(clique(0, 1), clique(2, 3, 4, 5)...)
	assign, _ := Detect(6, edges)
	assert.Equal(t, 0, assign[2], "the 4-node clique is community 0")
	assert.Equal(t, 1, assign[0])
}

func TestEmptyAndIsolated(t *testing.T) {
	assign, q := Detect(0, nil)
	assert.Empty(t, assign)
	assert.Equal(t, 0.0, q)

	assign, q = Detect(3, nil)
	assert.Equal(t, []int{0, 1, 2}, assign, "isolated nodes stay apart")
	assert.Equal(t, 0.0, q)
}

func TestInvalidEdgesIgnored(t *testing.T) {
	assign, _ := Detect(2, []Edge{{From: 0, To: 0, Weight: 5}, {From: 0, To: 9, Weight: 1}, {From: -1, To: 1, Weight: 1}})
	assert.Equal(t, []int{0, 1}, assign)
}

func TestDeterministic(t *testing.T) {
	edges := append(clique(0, 1, 2), clique(3, 4, 5)...)
	first, q1 := Detect(6, edges)
	for range 5 {
		again, q2 := Detect(6, edges)
		assert.Equal(t, first, again)
		assert.Equal(t, q1, q2)
	}
}
