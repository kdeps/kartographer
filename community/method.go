package community

import (
	"fmt"
	"sort"
)

// Detector assigns each of n nodes to a community and returns the
// partition's modularity, as Detect and Leiden do.
type Detector func(n int, edges []Edge) (assign []int, modularity float64)

// Method returns the detector a caller names: "louvain" (Detect) or "leiden"
// (Leiden, whose communities are always connected).
func Method(name string) (Detector, error) {
	methods := map[string]Detector{"louvain": Detect, "leiden": Leiden}
	if d, ok := methods[name]; ok {
		return d, nil
	}
	names := make([]string, 0, len(methods))
	for n := range methods {
		names = append(names, n)
	}
	sort.Strings(names)
	return nil, fmt.Errorf("unknown community method %q (want one of %v)", name, names)
}
