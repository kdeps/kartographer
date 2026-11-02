package community

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// twoCliques joins nodes 0-3 and 4-7 each as a clique, with one bridge edge.
func twoCliques() []Edge {
	var edges []Edge
	for _, base := range []int{0, 4} {
		for i := 0; i < 4; i++ {
			for j := i + 1; j < 4; j++ {
				edges = append(edges, Edge{From: base + i, To: base + j, Weight: 1})
			}
		}
	}
	return append(edges, Edge{From: 3, To: 4, Weight: 1})
}

func TestLeidenFindsTheTwoCliques(t *testing.T) {
	assign, q := Leiden(8, twoCliques())
	assert.Equal(t, assign[0], assign[1])
	assert.Equal(t, assign[0], assign[3])
	assert.Equal(t, assign[4], assign[7])
	assert.NotEqual(t, assign[0], assign[4])
	assert.Greater(t, q, 0.0)
}

func TestLeidenAgreesWithLouvainOnCleanCliques(t *testing.T) {
	louvain, lq := Detect(8, twoCliques())
	leiden, eq := Leiden(8, twoCliques())
	assert.Equal(t, louvain, leiden)
	assert.InDelta(t, lq, eq, 1e-9)
}

func TestSplitDisconnectedSeparatesUnconnectedParts(t *testing.T) {
	// Two edges, 0-1 and 2-3, both in community 7: they are two parts.
	adj := make([]map[int]float64, 4)
	for i := range adj {
		adj[i] = map[int]float64{}
	}
	adj[0][1], adj[1][0] = 1, 1
	adj[2][3], adj[3][2] = 1, 1
	got := splitDisconnected(adj, []int{7, 7, 7, 7})
	assert.Equal(t, []int{0, 0, 1, 1}, got)
}

func TestSplitDisconnectedKeepsConnectedCommunityWhole(t *testing.T) {
	adj := make([]map[int]float64, 3)
	for i := range adj {
		adj[i] = map[int]float64{}
	}
	adj[0][1], adj[1][0] = 1, 1
	adj[1][2], adj[2][1] = 1, 1
	assert.Equal(t, []int{0, 0, 0}, splitDisconnected(adj, []int{5, 5, 5}))
}

func TestSplitDisconnectedIgnoresEdgesToOtherCommunities(t *testing.T) {
	adj := make([]map[int]float64, 2)
	for i := range adj {
		adj[i] = map[int]float64{}
	}
	adj[0][1], adj[1][0] = 1, 1
	assert.Equal(t, []int{0, 1}, splitDisconnected(adj, []int{1, 2}), "an edge across communities does not join them")
}
