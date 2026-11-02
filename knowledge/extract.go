package knowledge

import (
	"encoding/json"
	"errors"
	"slices"
	"strings"
)

// Entity is a thing a note is about, as a model named it.
type Entity struct {
	Name string `json:"name"`
	Kind string `json:"kind"`
}

// Relation is a link between two entities a model found.
type Relation struct {
	From       string  `json:"from"`
	To         string  `json:"to"`
	Relation   string  `json:"relation"`
	Confidence float64 `json:"confidence"`
}

// Found is what a model found in a note.
type Found struct {
	Entities  []Entity   `json:"entities"`
	Relations []Relation `json:"relations"`
}

// ParseOptions say which entities a reply may name. The caller owns them.
type ParseOptions struct {
	Kinds       []string // the entity kinds kept as named
	DefaultKind string   // the kind of an entity outside Kinds
	MaxNameLen  int      // longer names are dropped
}

// ParseFound reads the first JSON object of a model's reply and keeps only
// well-formed entities and relations: named entities, and relations between
// two different entities of the reply, with the confidence held to 0..1. A
// reply cut off mid-list is closed after its last complete item.
func ParseFound(reply string, o ParseOptions) (Found, error) {
	start := strings.Index(reply, "{")
	if start < 0 {
		return Found{}, errors.New("reply holds no JSON object")
	}
	// Small models add text (or a second object) after the answer: decode the
	// first complete object only.
	var raw Found
	if err := json.NewDecoder(strings.NewReader(reply[start:])).Decode(&raw); err != nil {
		var ok bool
		if raw, ok = salvage(reply[start:]); !ok {
			return Found{}, err
		}
	}
	var out Found
	names := map[string]bool{}
	for _, e := range raw.Entities {
		e.Name, e.Kind = strings.TrimSpace(e.Name), strings.ToLower(strings.TrimSpace(e.Kind))
		if e.Name == "" || len(e.Name) > o.MaxNameLen {
			continue
		}
		if !slices.Contains(o.Kinds, e.Kind) {
			e.Kind = o.DefaultKind
		}
		out.Entities = append(out.Entities, e)
		names[strings.ToLower(e.Name)] = true
	}
	for _, r := range raw.Relations {
		r.From, r.To = strings.TrimSpace(r.From), strings.TrimSpace(r.To)
		if !names[strings.ToLower(r.From)] || !names[strings.ToLower(r.To)] || strings.EqualFold(r.From, r.To) {
			continue
		}
		r.Relation = strings.ToLower(strings.TrimSpace(r.Relation))
		r.Confidence = min(max(r.Confidence, 0), 1)
		out.Relations = append(out.Relations, r)
	}
	return out, nil
}

// salvage recovers a reply cut off mid-list (a model that ran out of tokens):
// it closes the JSON after the last complete item.
func salvage(s string) (Found, bool) {
	for i := strings.LastIndex(s, "}"); i > 0; i = strings.LastIndex(s[:i], "}") {
		for _, tail := range []string{"]}", "}"} {
			var f Found
			if json.Unmarshal([]byte(s[:i+1]+tail), &f) == nil {
				return f, true
			}
		}
	}
	return Found{}, false
}

// EntityID is the node id of an entity a model found.
func EntityID(name string) string { return "entity:" + strings.ToLower(name) }

// MergeOptions say how a note's found entities and relations join the graph.
// The caller owns them.
type MergeOptions struct {
	Mention           string   // the relation from the note to each entity
	MentionConfidence float64  // how sure a mention is
	Relations         []string // relations kept as the model named them
	Fallback          string   // the relation any other one becomes
	FallbackMax       float64  // the highest confidence a fallback relation keeps
	AmbiguousBelow    float64  // an inferred edge below this confidence is ambiguous
	// Evidence is the source location of the first line naming one of names;
	// nil leaves edges without one.
	Evidence func(names ...string) string
}

// AddFound adds what a model found in the note at path: each entity becomes
// the note it names (resolved through r) or an entity node shared by every
// note that mentions it, the note mentions each entity, and each relation joins
// two entities as an inferred edge.
func (b *Builder) AddFound(path string, f Found, r *Resolver, o MergeOptions) {
	evidence := o.Evidence
	if evidence == nil {
		evidence = func(...string) string { return "" }
	}
	ids := map[string]string{}
	names := map[string]string{}
	for _, e := range f.Entities {
		id := b.entityNode(e, r, path)
		ids[strings.ToLower(e.Name)] = id
		names[id] = e.Name
		b.Infer(NoteID(path), id, o.Mention, o.MentionConfidence, o.AmbiguousBelow, path, evidence(e.Name))
	}
	for _, rel := range f.Relations {
		from, to := ids[strings.ToLower(rel.From)], ids[strings.ToLower(rel.To)]
		if from == "" || to == "" {
			continue
		}
		name, conf := rel.Relation, rel.Confidence
		if !slices.Contains(o.Relations, name) {
			name, conf = o.Fallback, min(conf, o.FallbackMax)
		}
		b.Infer(from, to, name, conf, o.AmbiguousBelow, path, evidence(names[from], names[to]))
	}
}

// entityNode is the node for an entity: the note of the same name, or an
// entity node shared by every note that mentions it.
func (b *Builder) entityNode(e Entity, r *Resolver, from string) string {
	if p := r.Wiki(e.Name, from); p != "" {
		return NoteID(p)
	}
	id := EntityID(e.Name)
	b.Add(&Node{ID: id, Label: e.Name, Kind: e.Kind})
	return id
}
