package knowledge

import (
	"fmt"
	"path"
	"strings"
)

// Relations between notes, and the kinds of the nodes a document adds.
const (
	RelReferences = "references"
	RelRelatesTo  = "relates_to"
	RelDependsOn  = "depends_on"

	KindNote = "note"
)

// NoteID is the node id of the note at path.
func NoteID(path string) string { return "note:" + path }

// TagID is the node id of a tag. Tags are matched without case.
func TagID(tag string) string { return "tag:" + strings.ToLower(tag) }

// CodeID is the node id of a code package.
func CodeID(pkg string) string { return "code:" + pkg }

// LineRef is the source location text of a 1-based line, "" when unknown.
func LineRef(n int) string {
	if n <= 0 {
		return ""
	}
	return fmt.Sprintf("L%d", n)
}

// Doc is one note as the graph reads it: its node fields and the targets of
// its links and tags, with the line each was first seen on.
type Doc struct {
	Path      string // slash path, the note's identity
	Title     string
	Kind      string // KindNote when empty
	Summary   string
	Tags      []string
	Aliases   []string
	Links     []string // wiki link targets in the body
	MetaLinks []string // wiki link targets in the front matter
	MDLinks   []string // relative markdown link targets
	LinkLine  map[string]int
	MetaLine  map[string]int
	MDLine    map[string]int
	TagLine   map[string]int
}

// AddDoc adds d's node, then its wiki links (references from the body,
// relates_to from the front matter), its markdown links and its tags. A wiki
// target that resolves to no note is returned, so the caller can report it.
func (b *Builder) AddDoc(d Doc, r *Resolver) (missing []string) {
	kind := d.Kind
	if kind == "" {
		kind = KindNote
	}
	id := NoteID(d.Path)
	b.Add(&Node{ID: id, Label: d.Title, Kind: kind, Note: d.Path, Summary: d.Summary, Tags: d.Tags, Aliases: d.Aliases})
	missing = append(missing, b.linkWiki(id, d.Path, d.Links, d.LinkLine, RelReferences, r)...)
	missing = append(missing, b.linkWiki(id, d.Path, d.MetaLinks, d.MetaLine, RelRelatesTo, r)...)
	for _, t := range d.MDLinks {
		if target := r.Relative(t, d.Path); target != "" {
			b.Link(id, NoteID(target), RelReferences, d.Path, LineRef(d.MDLine[t]))
		}
	}
	for _, t := range d.Tags {
		tag := b.Add(&Node{ID: TagID(t), Label: "#" + t, Kind: KindTag})
		b.Link(id, tag.ID, RelTagged, d.Path, LineRef(d.TagLine[t]))
	}
	return missing
}

func (b *Builder) linkWiki(id, from string, targets []string, lines map[string]int, rel string, r *Resolver) (missing []string) {
	for _, t := range targets {
		if target := r.Wiki(t, from); target != "" {
			b.Link(id, NoteID(target), rel, from, LineRef(lines[t]))
		} else {
			missing = append(missing, t)
		}
	}
	return missing
}

// Resolver finds the note a link names: by path, by path suffix, by title or
// alias, the way markdown note apps resolve wiki links.
type Resolver struct {
	paths   map[string]string   // lower path without its extension -> path
	titles  map[string][]string // lower title or alias -> paths
	ordered []string            // the lower path keys, in input order
}

// NewResolver indexes docs by path, title and alias.
func NewResolver(docs []Doc) *Resolver {
	r := &Resolver{paths: map[string]string{}, titles: map[string][]string{}}
	for _, d := range docs {
		key := strings.ToLower(strings.TrimSuffix(d.Path, path.Ext(d.Path)))
		r.paths[key] = d.Path
		r.ordered = append(r.ordered, key)
		r.titles[strings.ToLower(d.Title)] = append(r.titles[strings.ToLower(d.Title)], d.Path)
		for _, a := range d.Aliases {
			r.titles[strings.ToLower(a)] = append(r.titles[strings.ToLower(a)], d.Path)
		}
	}
	return r
}

// Wiki resolves a wiki link target from the note at from, or "" when nothing
// matches. Among notes with the same title, the one closest to from wins.
func (r *Resolver) Wiki(target, from string) string {
	t := strings.ToLower(strings.TrimSuffix(strings.TrimSpace(target), ".md"))
	if t == "" {
		return ""
	}
	if p, ok := r.paths[t]; ok {
		return p
	}
	if strings.Contains(t, "/") {
		for _, k := range r.ordered {
			if strings.HasSuffix(k, "/"+t) {
				return r.paths[k]
			}
		}
		t = path.Base(t)
	}
	cands := r.titles[t]
	if len(cands) == 0 {
		return ""
	}
	best, bestShared := cands[0], -1
	for _, c := range cands {
		if s := sharedPrefix(path.Dir(c), path.Dir(from)); s > bestShared {
			best, bestShared = c, s
		}
	}
	return best
}

// Relative resolves a relative markdown link from the note at from, or "".
func (r *Resolver) Relative(target, from string) string {
	p := path.Clean(path.Join(path.Dir(from), target))
	key := strings.ToLower(strings.TrimSuffix(p, path.Ext(p)))
	return r.paths[key]
}

func sharedPrefix(a, b string) int {
	as, bs := strings.Split(a, "/"), strings.Split(b, "/")
	n := 0
	for n < len(as) && n < len(bs) && as[n] == bs[n] {
		n++
	}
	return n
}
