package search

// Interleave merges ranked lists into one: the first item of each list, then
// the second, and so on, up to limit items. An item whose key was already
// taken is left out, so a document two lists found appears once, at its best
// rank.
func Interleave[T any](lists [][]T, limit int, key func(T) string) []T {
	out := []T{}
	seen := map[string]bool{}
	for i := 0; len(out) < limit; i++ {
		more := false
		for _, list := range lists {
			if i >= len(list) {
				continue
			}
			more = true
			k := key(list[i])
			if seen[k] {
				continue
			}
			seen[k] = true
			out = append(out, list[i])
			if len(out) == limit {
				break
			}
		}
		if !more {
			break
		}
	}
	return out
}
