package knowledge

// KindTag is the kind of a node that stands for a tag. A note carrying a tag
// reaches the other notes with that tag through it.
const KindTag = "tag"

// Neighbour is a node connected to another node, with how strongly and by what path.
type Neighbour struct {
	Node     *Node
	Weight   float64 // EdgeWeight of the path's first edge, times any tag weight
	Chain    string  // the path as text, as Hit.Chain prints it
	Relation string
}

// Node returns the node with the given id, or nil.
func (ix *Index) Node(id string) *Node {
	if ix == nil {
		return nil
	}
	return ix.byID[id]
}

// Neighbours lists the notes connected to the node id, strongest edge first. A
// tag node expands to the other notes carrying it, reached by tag edges. Nodes
// that are neither notes nor tags are skipped, and id itself is never listed.
func (ix *Index) Neighbours(id string) []Neighbour {
	if ix == nil {
		return nil
	}
	var out []Neighbour
	for _, e := range ix.adj[id] {
		other := ix.byID[otherEnd(e, id)]
		if other == nil {
			continue
		}
		if other.Note != "" {
			out = append(out, Neighbour{Node: other, Weight: ix.edgeWeight(e),
				Chain: edgeChain(e), Relation: e.Relation})
			continue
		}
		if other.Kind != KindTag {
			continue
		}
		for _, te := range ix.adj[other.ID] {
			nn := ix.byID[otherEnd(te, other.ID)]
			if nn == nil || nn.Note == "" || nn.ID == id {
				continue
			}
			out = append(out, Neighbour{Node: nn, Weight: ix.edgeWeight(e),
				Chain: tagChain(id, other, nn), Relation: RelTagged})
		}
	}
	return out
}

func edgeChain(e *Edge) string {
	return Hit{Path: []PathStep{{From: e.From, To: e.To, Relation: e.Relation}}}.Chain()
}

func tagChain(from string, tag, to *Node) string {
	return Hit{Path: []PathStep{
		{From: from, To: tag.ID, Relation: RelTagged},
		{From: tag.ID, To: to.ID, Relation: RelTagged},
	}}.Chain()
}
