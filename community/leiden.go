package community

// Leiden is Detect with the connectivity guarantee of the Leiden method: after
// each pass, every community is split into its connected parts, so no
// community holds nodes that cannot reach each other inside it. Louvain can
// leave such communities. Communities are numbered as in Detect.
func Leiden(n int, edges []Edge) (assign []int, modularity float64) {
	return detect(n, edges, splitDisconnected)
}

// splitDisconnected gives each connected part of every community its own id.
// Two nodes are connected when a path of positive-weight edges joins them
// inside their community. The ids are assigned in node order, so the result
// is deterministic.
func splitDisconnected(adj []map[int]float64, comm []int) []int {
	out := make([]int, len(comm))
	for i := range out {
		out[i] = -1
	}
	next := 0
	for start := range comm {
		if out[start] != -1 {
			continue
		}
		out[start] = next
		stack := []int{start}
		for len(stack) > 0 {
			v := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			for u, w := range adj[v] {
				if u != v && w > 0 && comm[u] == comm[v] && out[u] == -1 {
					out[u] = next
					stack = append(stack, u)
				}
			}
		}
		next++
	}
	return out
}
