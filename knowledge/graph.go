// Package knowledge holds a provenance-tracked graph: nodes that are things,
// and edges that say how they relate and how the graph came to know it. An
// edge written in the source is extracted; an edge a model proposed is inferred
// with a confidence, or ambiguous when that confidence is too low.
package knowledge

import (
	"sort"

	"github.com/kdeps/kartographer/community"
)

// Provenance values of an edge.
const (
	ProvExtracted = "extracted"
	ProvInferred  = "inferred"
	ProvAmbiguous = "ambiguous"
)

// Node is one thing the graph knows.
type Node struct {
	ID        string   `json:"id"`
	Label     string   `json:"label"`
	Kind      string   `json:"kind"`
	Note      string   `json:"note,omitempty"`
	Summary   string   `json:"summary,omitempty"`
	Tags      []string `json:"tags,omitempty"`
	Aliases   []string `json:"aliases,omitempty"`
	Community int      `json:"community"`
}

// Edge is one relation between two nodes.
type Edge struct {
	From           string  `json:"from"`
	To             string  `json:"to"`
	Relation       string  `json:"relation"`
	Provenance     string  `json:"provenance"`
	Confidence     float64 `json:"confidence"`
	SourceFile     string  `json:"source_file,omitempty"`
	SourceLocation string  `json:"source_location,omitempty"`
}

// Graph is a set of nodes and edges, with the community structure found in it.
type Graph struct {
	Communities int      `json:"communities"`
	Modularity  float64  `json:"modularity"`
	Nodes       []*Node  `json:"nodes"`
	Edges       []*Edge  `json:"edges"`
	Missing     []string `json:"missing,omitempty"`
}

// Builder collects nodes and edges. Merge rules: an extracted edge is never
// replaced; a model's guess replaces an earlier one only when it is more
// confident; an edge from a node to itself is dropped.
type Builder struct {
	nodes map[string]*Node
	edges map[string]*Edge
}

// NewBuilder returns an empty builder.
func NewBuilder() *Builder {
	return &Builder{nodes: map[string]*Node{}, edges: map[string]*Edge{}}
}

// Add records a node. A node with the same ID is kept as it was first added.
func (b *Builder) Add(n *Node) *Node {
	if existing, ok := b.nodes[n.ID]; ok {
		return existing
	}
	b.nodes[n.ID] = n
	return n
}

// Link adds an extracted edge. The first extracted evidence wins.
func (b *Builder) Link(from, to, rel, file, loc string) {
	if from == to {
		return
	}
	key := from + "\x00" + to + "\x00" + rel
	if e, ok := b.edges[key]; ok && e.Provenance == ProvExtracted {
		return
	}
	b.edges[key] = &Edge{
		From: from, To: to, Relation: rel, Provenance: ProvExtracted, Confidence: 1,
		SourceFile: file, SourceLocation: loc,
	}
}

// Infer adds an edge a model proposed, with its confidence. It is ambiguous
// when the confidence is below ambiguousBelow. It does not replace an extracted
// edge, and a more confident guess replaces a less confident one.
func (b *Builder) Infer(from, to, rel string, conf, ambiguousBelow float64, file, loc string) {
	if from == to {
		return
	}
	prov := ProvInferred
	if conf < ambiguousBelow {
		prov = ProvAmbiguous
	}
	key := from + "\x00" + to + "\x00" + rel
	if e, ok := b.edges[key]; ok && (e.Provenance == ProvExtracted || e.Confidence >= conf) {
		return
	}
	b.edges[key] = &Edge{
		From: from, To: to, Relation: rel, Provenance: prov, Confidence: conf,
		SourceFile: file, SourceLocation: loc,
	}
}

// Graph returns the nodes and edges, sorted by ID (nodes) and by from, to and
// relation (edges). The community fields are not yet set.
func (b *Builder) Graph() *Graph {
	g := &Graph{}
	for _, n := range b.nodes {
		g.Nodes = append(g.Nodes, n)
	}
	for _, e := range b.edges {
		g.Edges = append(g.Edges, e)
	}
	sort.Slice(g.Nodes, func(i, j int) bool { return g.Nodes[i].ID < g.Nodes[j].ID })
	sort.Slice(g.Edges, func(i, j int) bool {
		a, b := g.Edges[i], g.Edges[j]
		if a.From != b.From {
			return a.From < b.From
		}
		if a.To != b.To {
			return a.To < b.To
		}
		return a.Relation < b.Relation
	})
	return g
}

// Communities groups the graph's nodes and sets each node's Community and the
// graph's Communities and Modularity. weight gives each edge's strength in the
// grouping. It returns the modularity.
func Communities(g *Graph, weight func(*Edge) float64) float64 {
	return CommunitiesWith(g, weight, community.Detect)
}

// CommunitiesWith is Communities with the detector the caller chooses, such as
// one from community.Method.
func CommunitiesWith(g *Graph, weight func(*Edge) float64, detect community.Detector) float64 {
	n := len(g.Nodes)
	if n == 0 {
		return 0
	}
	order := make([]int, n)
	for i := range order {
		order[i] = i
	}
	sort.Slice(order, func(a, b int) bool { return g.Nodes[order[a]].ID < g.Nodes[order[b]].ID })
	pos := make([]int, n)
	for p, i := range order {
		pos[i] = p
	}
	idx := make(map[string]int, n)
	for i, nd := range g.Nodes {
		idx[nd.ID] = pos[i]
	}
	var edges []community.Edge
	for _, e := range g.Edges {
		a, okA := idx[e.From]
		b, okB := idx[e.To]
		if !okA || !okB {
			continue
		}
		edges = append(edges, community.Edge{From: a, To: b, Weight: weight(e)})
	}
	assign, q := detect(n, edges)
	seen := map[int]bool{}
	for p, i := range order {
		g.Nodes[i].Community = assign[p]
		seen[assign[p]] = true
	}
	g.Communities = len(seen)
	g.Modularity = q
	return q
}
