package search

import (
	"math"
	"sort"
)

// Jaccard is the share of words a and b have in common: the size of their
// intersection over the size of their union. It is 0 when either is empty.
func Jaccard(a, b map[string]bool) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	common := 0
	for w := range a {
		if b[w] {
			common++
		}
	}
	return float64(common) / float64(len(a)+len(b)-common)
}

// Cosine is the cosine of the angle between a and b. It is 0 when their
// lengths differ, either is empty, or either has zero length as a vector.
func Cosine(a, b []float32) float64 {
	if len(a) == 0 || len(b) != len(a) {
		return 0
	}
	var dot, na, nb float64
	for i := range a {
		x, y := float64(a[i]), float64(b[i])
		dot += x * y
		na += x * x
		nb += y * y
	}
	d := math.Sqrt(na) * math.Sqrt(nb)
	if d == 0 {
		return 0
	}
	return dot / d
}

// TopK is the indices of the k highest scores, in index order. Equal scores
// keep the earlier index first. k of 0 or less, or k past len(scores), keeps
// every index.
func TopK(scores []float64, k int) []int {
	idx := make([]int, len(scores))
	for i := range idx {
		idx[i] = i
	}
	if k <= 0 || k >= len(idx) {
		return idx
	}
	sort.SliceStable(idx, func(i, j int) bool { return scores[idx[i]] > scores[idx[j]] })
	idx = idx[:k]
	sort.Ints(idx)
	return idx
}
