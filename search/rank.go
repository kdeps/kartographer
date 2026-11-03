package search

import "sort"

// Rank orders results the way Search does: more matched query terms first,
// then the higher score.
func Rank(results []SearchResult) {
	sort.Slice(results, func(i, j int) bool {
		if results[i].MatchCount != results[j].MatchCount {
			return results[i].MatchCount > results[j].MatchCount
		}
		return results[i].Score > results[j].Score
	})
}

// Boost multiplies each result's score by up to factor, scaled by
// weight(path) clamped to [0, 1]: weight 1 applies the whole factor, weight 0
// leaves the score alone. It then re-ranks the results.
func Boost(results []SearchResult, factor float64, weight func(path string) float64) {
	for i := range results {
		w := weight(results[i].Document.Path)
		if w <= 0 {
			continue
		}
		if w > 1 {
			w = 1
		}
		results[i].Score *= 1 + (factor-1)*w
	}
	Rank(results)
}
