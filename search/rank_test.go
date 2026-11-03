package search

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func result(path string, matches int, score float64) SearchResult {
	return SearchResult{Document: &Document{Path: path}, MatchCount: matches, Score: score}
}

func paths(rs []SearchResult) []string {
	out := make([]string, len(rs))
	for i, r := range rs {
		out[i] = r.Document.Path
	}
	return out
}

func TestRank(t *testing.T) {
	rs := []SearchResult{result("a", 1, 9), result("b", 2, 1), result("c", 2, 5)}
	Rank(rs)
	assert.Equal(t, []string{"c", "b", "a"}, paths(rs), "match count first, then score")
}

func TestBoost(t *testing.T) {
	rs := []SearchResult{result("a", 1, 1.0), result("b", 1, 0.9), result("c", 1, 0.8)}
	Boost(rs, 1.25, func(p string) float64 {
		return map[string]float64{"b": 1, "c": 0.5, "a": -1}[p]
	})
	assert.Equal(t, []string{"b", "a", "c"}, paths(rs))
	assert.InDelta(t, 1.125, rs[0].Score, 1e-9, "weight 1 applies the whole factor")
	assert.InDelta(t, 1.0, rs[1].Score, 1e-9, "a weight of 0 or less leaves the score")
	assert.InDelta(t, 0.9, rs[2].Score, 1e-9, "weight 0.5 applies half the boost")

	rs = []SearchResult{result("x", 1, 1)}
	Boost(rs, 2, func(string) float64 { return 5 })
	assert.InDelta(t, 2.0, rs[0].Score, 1e-9, "a weight above 1 counts as 1")
}
