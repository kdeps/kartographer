package knowledge

import (
	"fmt"
	"math"
	"slices"
	"sort"
	"strings"

	"github.com/kdeps/kartographer/text"
)

// RelTagged is the relation of an edge from a note to a tag it carries.
const RelTagged = "tagged"

// Tuning holds the numbers of a recall. The caller sets them, so the package
// holds no policy of its own.
type Tuning struct {
	MaxHops         int     // hops a walk takes from a seed; a larger or zero request is clamped to it
	DefaultBudget   int     // tokens of hits when a request gives none
	MaxSeeds        int     // best-matching nodes a request starts from
	MaxHits         int     // notes a recall returns at most
	BytesPerToken   int     // bytes per token when a hit's cost is estimated
	HopDecay        float64 // score kept per hop, before the edge weight and the node's fan-out
	LabelMatchBoost float64 // multiplier of a query word that is in the node's label
	FullLabelBoost  float64 // multiplier when the whole label appears in the query
	TagEdgeWeight   float64 // weight of a tag edge, against 1 for a link
	AmbiguousWeight float64 // weight of an ambiguous edge, times its confidence
}

// Options configure an Index.
type Options struct {
	Words  text.Options // how query and node text split into matchable words
	Tuning Tuning
	// NodeWeight scales a node's score by its source. Nil weighs every node 1.
	NodeWeight func(*Node) float64
}

// PathStep is one edge on the path that reached a recalled note.
type PathStep struct {
	From           string  `json:"from"`
	To             string  `json:"to"`
	Relation       string  `json:"relation"`
	Provenance     string  `json:"provenance"`
	Confidence     float64 `json:"confidence"`
	SourceFile     string  `json:"source_file,omitempty"`
	SourceLocation string  `json:"source_location,omitempty"`
}

// Hit is one recalled note.
type Hit struct {
	Note    string     `json:"note"` // the note's path, as the graph records it
	Title   string     `json:"title"`
	Summary string     `json:"summary,omitempty"`
	Score   float64    `json:"score"`
	Hops    int        `json:"hops"`
	Via     string     `json:"via,omitempty"` // the seed it was reached from
	Path    []PathStep `json:"path,omitempty"`
}

// Recall is the outcome of one recall: the notes worth reading for a request,
// best first, within the token budget.
type Recall struct {
	Seeds  []string `json:"seeds"`
	Hits   []Hit    `json:"hits"`
	Tokens int      `json:"tokens"`
}

// Index answers recalls over a loaded graph.
type Index struct {
	g     *Graph
	opt   Options
	terms map[string][]string // node id -> its tokens
	label map[string]map[string]bool
	idf   map[string]float64
	adj   map[string][]*Edge
	byID  map[string]*Node
}

// NewIndex prepares g for recall under o.
func NewIndex(g *Graph, o Options) *Index {
	ix := &Index{
		g: g, opt: o, terms: map[string][]string{}, label: map[string]map[string]bool{},
		idf: map[string]float64{}, adj: map[string][]*Edge{}, byID: map[string]*Node{},
	}
	df := map[string]int{}
	for _, n := range g.Nodes {
		ix.byID[n.ID] = n
		labelWords := ix.words(n.Label + " " + strings.Join(n.Aliases, " "))
		ix.label[n.ID] = text.Set(labelWords)
		toks := append(append(labelWords, ix.words(n.Summary)...), ix.words(strings.Join(n.Tags, " "))...)
		ix.terms[n.ID] = toks
		for t := range text.Set(toks) {
			df[t]++
		}
	}
	total := float64(len(g.Nodes)) + 1
	for t, d := range df {
		ix.idf[t] = math.Log(total / float64(d))
	}
	for _, e := range g.Edges {
		ix.adj[e.From] = append(ix.adj[e.From], e)
		ix.adj[e.To] = append(ix.adj[e.To], e)
	}
	for from, edges := range ix.adj {
		ix.sortAdj(from, edges)
	}
	return ix
}

func (ix *Index) words(s string) []string { return text.Words(s, ix.opt.Words) }

// sortAdj orders edges strongest first. Ties break on provenance, relation,
// the other node, then source file and location, so recall does not depend on
// the order edges were serialized.
func (ix *Index) sortAdj(from string, edges []*Edge) {
	sort.SliceStable(edges, func(i, j int) bool {
		wi, wj := ix.edgeWeight(edges[i]), ix.edgeWeight(edges[j])
		if wi != wj {
			return wi > wj
		}
		if provOrder(edges[i].Provenance) != provOrder(edges[j].Provenance) {
			return provOrder(edges[i].Provenance) < provOrder(edges[j].Provenance)
		}
		if edges[i].Relation != edges[j].Relation {
			return edges[i].Relation < edges[j].Relation
		}
		oi, oj := otherEnd(edges[i], from), otherEnd(edges[j], from)
		if oi != oj {
			return oi < oj
		}
		if edges[i].SourceFile != edges[j].SourceFile {
			return edges[i].SourceFile < edges[j].SourceFile
		}
		return edges[i].SourceLocation < edges[j].SourceLocation
	})
}

func otherEnd(e *Edge, from string) string {
	if e.From == from {
		return e.To
	}
	return e.From
}

const (
	provRankExtracted = iota
	provRankInferred
	provRankAmbiguous
	provRankOther
)

func provOrder(p string) int {
	switch p {
	case ProvExtracted:
		return provRankExtracted
	case ProvInferred:
		return provRankInferred
	case ProvAmbiguous:
		return provRankAmbiguous
	default:
		return provRankOther
	}
}

// EdgeWeight is how strongly an edge carries a recall: an extracted edge
// counts 1, an inferred edge its confidence, an ambiguous edge AmbiguousWeight
// times its confidence, and a tag edge TagEdgeWeight of that.
func EdgeWeight(e *Edge, t Tuning) float64 {
	w := 1.0
	switch e.Provenance {
	case ProvInferred:
		w = e.Confidence
	case ProvAmbiguous:
		w = t.AmbiguousWeight * e.Confidence
	}
	if e.Relation == RelTagged {
		w *= t.TagEdgeWeight
	}
	return w
}

func (ix *Index) edgeWeight(e *Edge) float64 { return EdgeWeight(e, ix.opt.Tuning) }

// Recall finds the notes related to query: it scores nodes on the request's
// words, walks up to hops edges from the best ones (each hop multiplies by
// HopDecay and the edge's weight), strongest edge first, and keeps the best
// notes within budget tokens. No model is involved.
//
// Notes in exclude are never recalled.
func (ix *Index) Recall(query string, hops, budget int, exclude ...string) Recall {
	t := ix.opt.Tuning
	if hops <= 0 || hops > t.MaxHops {
		hops = t.MaxHops
	}
	if budget <= 0 {
		budget = t.DefaultBudget
	}
	best := map[string]Hit{}
	var r Recall
	for _, sd := range ix.seeds(query) {
		r.Seeds = append(r.Seeds, ix.byID[sd.id].Label)
		ix.walk(sd.id, sd.score, hops, best)
	}
	hits := make([]Hit, 0, len(best))
	for _, h := range best {
		hits = append(hits, h)
	}
	sort.Slice(hits, func(i, j int) bool {
		if hits[i].Score != hits[j].Score {
			return hits[i].Score > hits[j].Score
		}
		return hits[i].Note < hits[j].Note
	})
	for _, h := range hits {
		if slices.Contains(exclude, h.Note) {
			continue
		}
		extra := h.Chain()
		if extra == "" {
			extra = h.Via
		}
		cost := (len(h.Note) + len(h.Summary) + len(extra) + 16 + t.BytesPerToken - 1) / t.BytesPerToken
		if len(r.Hits) >= t.MaxHits || r.Tokens+cost > budget {
			break
		}
		r.Hits = append(r.Hits, h)
		r.Tokens += cost
	}
	return r
}

type scored struct {
	id    string
	score float64
}

// seeds are the best-matching nodes for query, best first.
func (ix *Index) seeds(query string) []scored {
	q := text.Set(ix.words(query))
	lowerQuery := strings.ToLower(query)
	var out []scored
	for id, toks := range ix.terms {
		if s := ix.score(id, toks, q, lowerQuery); s > 0 {
			out = append(out, scored{id, s})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].score != out[j].score {
			return out[i].score > out[j].score
		}
		return out[i].id < out[j].id
	})
	if len(out) > ix.opt.Tuning.MaxSeeds {
		out = out[:ix.opt.Tuning.MaxSeeds]
	}
	return out
}

// score weighs a node's words that the query shares (rarer words count more,
// label words are boosted) and boosts it again when the whole label appears.
func (ix *Index) score(id string, toks []string, q map[string]bool, lowerQuery string) float64 {
	var s float64
	for t := range text.Set(toks) {
		if !q[t] {
			continue
		}
		w := ix.idf[t]
		if ix.label[id][t] {
			w *= ix.opt.Tuning.LabelMatchBoost
		}
		s += w
	}
	label := strings.ToLower(strings.TrimPrefix(ix.byID[id].Label, "#"))
	// A label no longer than the shortest matchable word is too common to prove a match.
	if s > 0 && len(label) > ix.opt.Words.MinWordLength && strings.Contains(lowerQuery, label) {
		s *= ix.opt.Tuning.FullLabelBoost
	}
	return s * ix.nodeWeight(ix.byID[id])
}

func (ix *Index) nodeWeight(n *Node) float64 {
	if ix.opt.NodeWeight == nil {
		return 1
	}
	return ix.opt.NodeWeight(n)
}

// walk spreads score from a seed, strongest edge first. A node is expanded
// again when a higher score reaches it, so a weak edge listed first cannot hide
// a stronger path. The hop limit still bounds the walk.
func (ix *Index) walk(seed string, score float64, hops int, best map[string]Hit) {
	type item struct {
		id    string
		score float64
		hop   int
		path  []PathStep
	}
	bestAt := map[string]float64{seed: score}
	queue := []item{{id: seed, score: score}}
	seedLabel := ""
	if n := ix.byID[seed]; n != nil {
		seedLabel = n.Label
	}
	for len(queue) > 0 {
		bi := 0
		for i := 1; i < len(queue); i++ {
			if queue[i].score > queue[bi].score {
				bi = i
			}
		}
		it := queue[bi]
		queue = append(queue[:bi], queue[bi+1:]...)
		if it.score < bestAt[it.id] {
			continue
		}
		ix.keep(it.id, it.score, it.hop, seedLabel, it.path, best)
		if it.hop >= hops {
			continue
		}
		for _, e := range ix.adj[it.id] {
			other := otherEnd(e, it.id)
			next := it.score * ix.edgeWeight(e) * ix.opt.Tuning.HopDecay / math.Log(math.E+float64(len(ix.adj[it.id])))
			if prev, ok := bestAt[other]; ok && next <= prev {
				continue
			}
			bestAt[other] = next
			step := PathStep{
				From: it.id, To: other, Relation: e.Relation, Provenance: e.Provenance,
				Confidence: e.Confidence, SourceFile: e.SourceFile, SourceLocation: e.SourceLocation,
			}
			path := append(append([]PathStep{}, it.path...), step)
			queue = append(queue, item{other, next, it.hop + 1, path})
		}
	}
}

// keep records a reached note when it beats its earlier score, with the path
// that produced that score.
func (ix *Index) keep(id string, score float64, hop int, seedLabel string, path []PathStep, best map[string]Hit) {
	n := ix.byID[id]
	if n == nil || n.Note == "" {
		return
	}
	if h, ok := best[n.Note]; ok && score <= h.Score {
		return
	}
	via := ""
	if hop > 0 {
		via = seedLabel
	}
	best[n.Note] = Hit{
		Note: n.Note, Title: n.Label, Summary: n.Summary, Score: score, Hops: hop, Via: via, Path: path,
	}
}

// Chain is the relation path from the seed, for the model and for token accounting.
func (h Hit) Chain() string {
	if len(h.Path) == 0 {
		return ""
	}
	var b strings.Builder
	for i, s := range h.Path {
		if i == 0 {
			b.WriteString(shortID(s.From))
		}
		rel := s.Relation
		if rel == "" {
			rel = "link"
		}
		fmt.Fprintf(&b, " --%s--> %s", rel, shortID(s.To))
	}
	return b.String()
}

// shortID drops a lowercase kind prefix such as "note:" or "tag:" from a node id.
func shortID(id string) string {
	i := strings.IndexByte(id, ':')
	if i <= 0 {
		return id
	}
	for _, c := range id[:i] {
		if c < 'a' || c > 'z' {
			return id
		}
	}
	return id[i+1:]
}
