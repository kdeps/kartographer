package knowledge

import (
	"sort"
	"strings"
)

// ReportOptions sets how many entries each list of a report keeps.
type ReportOptions struct {
	Top          int // god nodes and surprising connections
	CommunityTop int // best-connected members listed per community
	MissingTop   int // missing link targets and low-confidence edges to review
}

// Report is what a graph says about itself.
type Report struct {
	Notes, Nodes, Edges, Extracted, Communities int
	Modularity                                  float64
	GodNodes                                    []Ranked
	Groups                                      []Group
	Surprising                                  []Bridge
	Missing                                     []string
	Review                                      []*Edge // ambiguous edges, most confident first
}

// Ranked is a node with its degree.
type Ranked struct {
	Node   *Node
	Degree int
}

// Group is one community: its size and best-connected members.
type Group struct {
	ID      int
	Size    int
	Top     []Ranked
	Summary string // the best-connected note's summary
}

// Bridge is a lone edge between two communities that share no other edge: a
// connection nobody would have looked for.
type Bridge struct {
	Edge               *Edge
	From, To           *Node
	FromGroup, ToGroup int
	FromSize, ToSize   int
}

// NewReport analyses g, whose communities must already be set.
func NewReport(g *Graph, modularity float64, o ReportOptions) Report {
	r := Report{Nodes: len(g.Nodes), Edges: len(g.Edges), Modularity: modularity, Missing: g.Missing}
	byID := map[string]*Node{}
	deg := map[string]int{}
	size := map[int]int{}
	for _, n := range g.Nodes {
		byID[n.ID] = n
		size[n.Community]++
		if n.Note != "" {
			r.Notes++
		}
	}
	r.Communities = len(size)
	for _, e := range g.Edges {
		deg[e.From]++
		deg[e.To]++
		switch e.Provenance {
		case ProvExtracted:
			r.Extracted++
		case ProvAmbiguous:
			r.Review = append(r.Review, e)
		}
	}
	sort.SliceStable(r.Review, func(i, j int) bool { return r.Review[i].Confidence > r.Review[j].Confidence })
	if len(r.Review) > o.MissingTop {
		r.Review = r.Review[:o.MissingTop]
	}
	ranked := make([]Ranked, 0, len(g.Nodes))
	for _, n := range g.Nodes {
		if n.Kind != KindTag {
			ranked = append(ranked, Ranked{n, deg[n.ID]})
		}
	}
	sort.SliceStable(ranked, func(i, j int) bool {
		if ranked[i].Degree != ranked[j].Degree {
			return ranked[i].Degree > ranked[j].Degree
		}
		return ranked[i].Node.ID < ranked[j].Node.ID
	})
	r.GodNodes = head(ranked, o.Top)
	r.Groups = groups(ranked, size, o.CommunityTop)
	r.Surprising = bridges(g, byID, size, o.Top)
	if len(r.Missing) > o.MissingTop {
		r.Missing = r.Missing[:o.MissingTop]
	}
	return r
}

func head(r []Ranked, n int) []Ranked {
	if len(r) > n {
		return r[:n]
	}
	return r
}

// groups lists communities largest first, each with its best-connected members
// (ranked is sorted by degree). Single-node communities are left out.
func groups(ranked []Ranked, size map[int]int, top int) []Group {
	byGroup := map[int]*Group{}
	for _, rk := range ranked {
		c := rk.Node.Community
		gr := byGroup[c]
		if gr == nil {
			gr = &Group{ID: c, Size: size[c]}
			byGroup[c] = gr
		}
		if len(gr.Top) < top {
			gr.Top = append(gr.Top, rk)
		}
		if gr.Summary == "" && rk.Node.Summary != "" {
			gr.Summary = rk.Node.Summary
		}
	}
	out := make([]Group, 0, len(byGroup))
	for _, gr := range byGroup {
		if gr.Size > 1 {
			out = append(out, *gr)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// bridges are edges between two communities joined by that edge alone, ranked
// by the smaller community's size: a lone link between two large groups is the
// least expected. Tag nodes are not part of any community's story, so they are skipped.
func bridges(g *Graph, byID map[string]*Node, size map[int]int, top int) []Bridge {
	type pair struct{ a, b int }
	count := map[pair]int{}
	var cands []Bridge
	for _, e := range g.Edges {
		from, to := byID[e.From], byID[e.To]
		if from == nil || to == nil || from.Community == to.Community || from.Kind == KindTag || to.Kind == KindTag {
			continue
		}
		p := pair{min(from.Community, to.Community), max(from.Community, to.Community)}
		count[p]++
		cands = append(cands, Bridge{
			Edge: e, From: from, To: to, FromGroup: from.Community, ToGroup: to.Community,
			FromSize: size[from.Community], ToSize: size[to.Community],
		})
	}
	var out []Bridge
	for _, b := range cands {
		if count[pair{min(b.FromGroup, b.ToGroup), max(b.FromGroup, b.ToGroup)}] == 1 && min(b.FromSize, b.ToSize) > 1 {
			out = append(out, b)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		si, sj := min(out[i].FromSize, out[i].ToSize), min(out[j].FromSize, out[j].ToSize)
		if si != sj {
			return si > sj
		}
		return out[i].Edge.From+out[i].Edge.To < out[j].Edge.From+out[j].Edge.To
	})
	if len(out) > top {
		out = out[:top]
	}
	return out
}

// MatchEdges lists the edges whose endpoints, labels or source file contain
// query, case-insensitively, in graph order. Nil for an empty query.
func MatchEdges(g *Graph, query string) []*Edge {
	q := strings.ToLower(strings.TrimSpace(query))
	if g == nil || q == "" {
		return nil
	}
	labels := map[string]string{}
	for _, n := range g.Nodes {
		labels[n.ID] = strings.ToLower(n.Label + " " + n.Note + " " + n.ID)
	}
	var out []*Edge
	for _, e := range g.Edges {
		blob := strings.ToLower(e.From + " " + e.To + " " + e.SourceFile + " " + labels[e.From] + " " + labels[e.To])
		if strings.Contains(blob, q) {
			out = append(out, e)
		}
	}
	return out
}
