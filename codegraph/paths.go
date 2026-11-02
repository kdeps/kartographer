package codegraph

import "sort"

// Limits for path and neighbourhood queries.
const (
	maxPathDepth = 8
	maxPaths     = 10
)

// Trace returns up to k simple directed call paths from src to tgt, shortest
// first. Each path lists symbol names from src to tgt.
func (ix *Index) Trace(src, tgt string, k int) ([][]string, error) {
	from, err := ix.resolve(src)
	if err != nil {
		return nil, err
	}
	to, err := ix.resolve(tgt)
	if err != nil {
		return nil, err
	}
	k = min(max(k, 1), maxPaths)
	var paths [][]string
	var walk func(cur string, path []string, on map[string]bool)
	walk = func(cur string, path []string, on map[string]bool) {
		if len(paths) >= k || len(path) > maxPathDepth {
			return
		}
		if cur == to {
			paths = append(paths, append([]string{}, path...))
			return
		}
		for _, next := range ix.deps.NodeDependencies[cur] {
			if !on[next] {
				on[next] = true
				walk(next, append(path, next), on)
				on[next] = false
			}
		}
	}
	walk(from, []string{from}, map[string]bool{from: true})
	sort.SliceStable(paths, func(i, j int) bool { return len(paths[i]) < len(paths[j]) })
	return paths, nil
}

// ShortestPath is the shortest connection between src and tgt when call edges
// are followed in either direction. It is nil when they are not connected.
func (ix *Index) ShortestPath(src, tgt string) ([]string, error) {
	from, err := ix.resolve(src)
	if err != nil {
		return nil, err
	}
	to, err := ix.resolve(tgt)
	if err != nil {
		return nil, err
	}
	prev := map[string]string{from: ""}
	queue := []string{from}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if cur == to {
			break
		}
		for _, next := range append(append([]string{}, ix.deps.NodeDependencies[cur]...), ix.callers[cur]...) {
			if _, seen := prev[next]; !seen {
				prev[next] = cur
				queue = append(queue, next)
			}
		}
	}
	if _, ok := prev[to]; !ok {
		return nil, nil
	}
	var path []string
	for n := to; n != ""; n = prev[n] {
		path = append([]string{n}, path...)
	}
	return path, nil
}

// Neighbourhood is the set of symbols within depth hops of the seeds, and the
// call edges between them, capped at nodeLimit nodes.
func (ix *Index) Neighbourhood(seeds []string, depth, nodeLimit int) (nodes []string, edges [][2]string) {
	depth = min(max(depth, 1), 3)
	seen := map[string]bool{}
	frontier := []string{}
	for _, s := range seeds {
		if name, err := ix.resolve(s); err == nil && !seen[name] {
			seen[name] = true
			frontier = append(frontier, name)
		}
	}
	for d := 0; d < depth && len(seen) < nodeLimit; d++ {
		var next []string
		for _, cur := range frontier {
			for _, n := range append(append([]string{}, ix.deps.NodeDependencies[cur]...), ix.callers[cur]...) {
				if !seen[n] && len(seen) < nodeLimit {
					seen[n] = true
					next = append(next, n)
				}
			}
		}
		frontier = next
	}
	for n := range seen {
		nodes = append(nodes, n)
	}
	sort.Strings(nodes)
	for _, from := range nodes {
		for _, to := range ix.deps.NodeDependencies[from] {
			if seen[to] {
				edges = append(edges, [2]string{from, to})
			}
		}
	}
	return nodes, edges
}

// Node is a symbol with its source and its direct callers and callees.
func (ix *Index) Node(symbol string, limit int) (Symbol, string, []Symbol, error) {
	name, err := ix.resolve(symbol)
	if err != nil {
		return Symbol{}, "", nil, err
	}
	sym := ix.Symbols[name]
	neigh, _ := ix.Neighbours(name, limit)
	return sym, ix.body(sym), neigh, nil
}

// Expand returns the source of each named symbol; names that do not resolve
// are reported with an empty body.
func (ix *Index) Expand(names []string) map[string]string {
	out := map[string]string{}
	for _, n := range names {
		if name, err := ix.resolve(n); err == nil {
			out[n] = ix.body(ix.Symbols[name])
		} else {
			out[n] = ""
		}
	}
	return out
}
