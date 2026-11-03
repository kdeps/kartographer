// Package community groups the nodes of a weighted undirected graph with the
// Louvain method: each node moves to the neighbouring community that most
// raises modularity, communities then become single nodes, and the moves
// repeat until nothing moves.
package community

import "sort"

// maxPasses bounds the aggregation passes (each one at least halves the work
// in practice).
const maxPasses = 32

const (
	// gainEpsilon ignores modularity gains lost in float rounding.
	gainEpsilon = 1e-12
	// bothEnds counts an undirected edge's weight once at each end.
	bothEnds = 2
)

// Edge joins two nodes by index with a weight. Self-loops and indices out of
// range are ignored.
type Edge struct {
	From, To int
	Weight   float64
}

// Detect assigns each of n nodes to a community and returns the partition's
// modularity. Communities are numbered 0 upward, largest first; ties go to
// the community holding the smallest node index. It is deterministic.
func Detect(n int, edges []Edge) (assign []int, modularity float64) {
	return detect(n, edges, nil)
}

// detect is Detect with an optional refinement applied to each pass's
// partition before it is aggregated. A nil refinement is Louvain.
func detect(n int, edges []Edge, refine func([]map[int]float64, []int) []int) (assign []int, modularity float64) {
	assign = make([]int, n)
	for i := range assign {
		assign[i] = i
	}
	if n == 0 {
		return assign, 0
	}
	adj := make([]map[int]float64, n)
	for i := range adj {
		adj[i] = map[int]float64{}
	}
	valid := make([]Edge, 0, len(edges))
	for _, e := range edges {
		if e.From < 0 || e.To < 0 || e.From >= n || e.To >= n || e.From == e.To {
			continue
		}
		adj[e.From][e.To] += e.Weight
		adj[e.To][e.From] += e.Weight
		valid = append(valid, e)
	}
	member := make([]int, n) // node -> current community
	for i := range member {
		member[i] = i
	}
	for range maxPasses {
		comm, moved := localMoves(adj)
		if !moved {
			break
		}
		if refine != nil {
			comm = refine(adj, comm)
		}
		comm, count := renumber(comm)
		for i := range member {
			member[i] = comm[member[i]]
		}
		adj = aggregate(adj, comm, count)
	}
	assign = orderBySize(member)
	return assign, modularityOf(n, valid, assign)
}

// localMoves is Louvain's first phase on one level's graph.
func localMoves(adj []map[int]float64) ([]int, bool) {
	n := len(adj)
	comm := make([]int, n)
	k := make([]float64, n)
	tot := make([]float64, n)
	var m2 float64
	for i := range adj {
		comm[i] = i
		for _, w := range adj[i] {
			k[i] += w
		}
		tot[i] = k[i]
		m2 += k[i]
	}
	if m2 == 0 {
		return comm, false
	}
	movedAny := false
	for changed := true; changed; {
		changed = false
		for i := range n {
			old := comm[i]
			tot[old] -= k[i]
			best := bestCommunity(i, adj[i], comm, tot, k[i], m2)
			comm[i] = best
			tot[best] += k[i]
			if best != old {
				changed, movedAny = true, true
			}
		}
	}
	return comm, movedAny
}

// bestCommunity is the neighbouring community (or node i's own) whose
// modularity gain for i is highest; ties keep the lower id.
func bestCommunity(i int, row map[int]float64, comm []int, tot []float64, ki, m2 float64) int {
	links := map[int]float64{}
	for j, w := range row {
		if j != i {
			links[comm[j]] += w
		}
	}
	old := comm[i]
	best, bestGain := old, links[old]-tot[old]*ki/m2
	cands := make([]int, 0, len(links))
	for c := range links {
		cands = append(cands, c)
	}
	sort.Ints(cands)
	for _, c := range cands {
		if gain := links[c] - tot[c]*ki/m2; gain > bestGain+gainEpsilon {
			best, bestGain = c, gain
		}
	}
	return best
}

// renumber maps community ids to 0..count-1 in first-seen order.
func renumber(comm []int) ([]int, int) {
	ids := map[int]int{}
	out := make([]int, len(comm))
	for i, c := range comm {
		id, ok := ids[c]
		if !ok {
			id = len(ids)
			ids[c] = id
		}
		out[i] = id
	}
	return out, len(ids)
}

// aggregate makes each community one node; internal weight becomes a
// self-loop, so k and modularity carry over.
func aggregate(adj []map[int]float64, comm []int, count int) []map[int]float64 {
	out := make([]map[int]float64, count)
	for i := range out {
		out[i] = map[int]float64{}
	}
	for i, row := range adj {
		for j, w := range row {
			out[comm[i]][comm[j]] += w
		}
	}
	return out
}

// orderBySize renumbers communities largest first (ties: the community
// holding the smallest node index first).
func orderBySize(member []int) []int {
	size := map[int]int{}
	first := map[int]int{}
	for i, c := range member {
		if _, seen := first[c]; !seen {
			first[c] = i
		}
		size[c]++
	}
	ids := make([]int, 0, len(size))
	for c := range size {
		ids = append(ids, c)
	}
	sort.Slice(ids, func(a, b int) bool {
		if size[ids[a]] != size[ids[b]] {
			return size[ids[a]] > size[ids[b]]
		}
		return first[ids[a]] < first[ids[b]]
	})
	rank := map[int]int{}
	for r, c := range ids {
		rank[c] = r
	}
	out := make([]int, len(member))
	for i, c := range member {
		out[i] = rank[c]
	}
	return out
}

// modularityOf is the modularity of assign over the valid edges.
func modularityOf(n int, edges []Edge, assign []int) float64 {
	k := make([]float64, n)
	var m2, in float64
	for _, e := range edges {
		k[e.From] += e.Weight
		k[e.To] += e.Weight
		m2 += bothEnds * e.Weight
		if assign[e.From] == assign[e.To] {
			in += bothEnds * e.Weight
		}
	}
	if m2 == 0 {
		return 0
	}
	// Communities are numbered below n; summing them in id order keeps the
	// float result the same on every call.
	tot := make([]float64, n)
	for i := range n {
		tot[assign[i]] += k[i]
	}
	q := in / m2
	for _, t := range tot {
		q -= (t / m2) * (t / m2)
	}
	return q
}
