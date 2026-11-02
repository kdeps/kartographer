package codegraph

import "sort"

// Affected lists the symbols that depend on the target through callers, up to
// depth hops. Each symbol appears once, at its nearest distance.
func (ix *Index) Affected(symbol string, depth, limit int) ([]Symbol, error) {
	name, err := ix.resolve(symbol)
	if err != nil {
		return nil, err
	}
	dist := map[string]int{name: 0}
	frontier := []string{name}
	for d := 1; d <= depth && len(frontier) > 0; d++ {
		var next []string
		for _, cur := range frontier {
			for _, c := range ix.callers[cur] {
				if _, seen := dist[c]; !seen {
					dist[c] = d
					next = append(next, c)
				}
			}
		}
		frontier = next
	}
	delete(dist, name)
	names := make([]string, 0, len(dist))
	for n := range dist {
		names = append(names, n)
	}
	sort.Slice(names, func(i, j int) bool {
		if dist[names[i]] != dist[names[j]] {
			return dist[names[i]] < dist[names[j]]
		}
		return names[i] < names[j]
	})
	out := make([]Symbol, 0, len(names))
	for _, n := range names {
		if sym, ok := ix.Symbols[n]; ok {
			out = append(out, sym)
		}
	}
	return capped(out, limit), nil
}
