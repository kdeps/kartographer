package graph

// Nearest walks breadth-first from start and returns up to max nodes, closest
// first, start included. next gives a node's successors, and ok false for a
// node that does not exist: such a node is neither returned nor walked past.
// max of 0 or less returns nothing.
func Nearest(start string, max int, next func(node string) (successors []string, ok bool)) []string {
	var out []string
	seen := map[string]bool{}
	queue := []string{start}
	for len(queue) > 0 && len(out) < max {
		n := queue[0]
		queue = queue[1:]
		if seen[n] {
			continue
		}
		seen[n] = true
		succ, ok := next(n)
		if !ok {
			continue
		}
		out = append(out, n)
		for _, s := range succ {
			if !seen[s] {
				queue = append(queue, s)
			}
		}
	}
	return out
}
