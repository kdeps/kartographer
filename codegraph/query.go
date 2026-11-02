package codegraph

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
)

// Keyword ranking: a symbol scores for each query word its name or file path
// contains, weighted by rules.yaml, plus a bonus for how connected it is. It is
// a lexical match, not a semantic one.

// Hit is a symbol that matched a question, with its source when requested.
type Hit struct {
	Symbol Symbol
	Score  float64
	Body   string
}

// Words are the lowercased words of a question, without stopwords.
func Words(question string) []string {
	fields := strings.FieldsFunc(strings.ToLower(question), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_'
	})
	var out []string
	for _, f := range fields {
		if len(f) > 1 && !stopwords[f] {
			out = append(out, f)
		}
	}
	return out
}

// score is how well a symbol matches the words.
func (ix *Index) score(sym Symbol, words []string) float64 {
	name := strings.ToLower(sym.Name)
	path := strings.ToLower(sym.File)
	var total float64
	for _, w := range words {
		if strings.Contains(name, w) {
			total += rules.Ranking.NameWeight
		}
		if strings.Contains(path, w) {
			total += rules.Ranking.PathWeight
		}
	}
	if total == 0 {
		return 0
	}
	degree := float64(len(ix.callers[sym.Name]) + len(ix.deps.NodeDependencies[sym.Name]))
	return total + rules.Ranking.DegreeWeight*degree
}

// Query returns the k best-matching symbols for question, with their source
// text cut to budget characters. skeleton drops the source.
func (ix *Index) Query(question string, k, budget int, skeleton bool) []Hit {
	words := Words(question)
	var hits []Hit
	for _, sym := range ix.Symbols {
		if sc := ix.score(sym, words); sc > 0 {
			hits = append(hits, Hit{Symbol: sym, Score: sc})
		}
	}
	sortHits(hits)
	if k > 0 && len(hits) > k {
		hits = hits[:k]
	}
	if skeleton {
		return hits
	}
	left := budget
	for i := range hits {
		body := ix.body(hits[i].Symbol)
		if len(body) > left {
			body = body[:max(left, 0)]
		}
		hits[i].Body = body
		left -= len(body)
	}
	return hits
}

// RankFiles scores each file by the symbols in it that match question.
func (ix *Index) RankFiles(question string, limit int) []Hit {
	words := Words(question)
	files := map[string]float64{}
	for _, sym := range ix.Symbols {
		if sc := ix.score(sym, words); sc > 0 {
			files[sym.File] += sc
		}
	}
	var hits []Hit
	for f, sc := range files {
		hits = append(hits, Hit{Symbol: Symbol{Name: f, File: f}, Score: sc})
	}
	sortHits(hits)
	if limit > 0 && len(hits) > limit {
		hits = hits[:limit]
	}
	return hits
}

// FileSymbols are the symbols defined in a file, in source order.
func (ix *Index) FileSymbols(file string) []Symbol {
	var out []Symbol
	for _, s := range ix.Symbols {
		if s.File == file {
			out = append(out, s)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Line < out[j].Line })
	return out
}

// FileNeighbors are the files this file calls into (dependencies) and the
// files that call into it (dependents), from the call graph.
func (ix *Index) FileNeighbors(file string, limit int) (dependencies, dependents []string) {
	deps := map[string]bool{}
	users := map[string]bool{}
	for _, s := range ix.FileSymbols(file) {
		for _, c := range ix.deps.NodeDependencies[s.Name] {
			if f := ix.Symbols[c].File; f != file {
				deps[f] = true
			}
		}
		for _, c := range ix.callers[s.Name] {
			if f := ix.Symbols[c].File; f != file {
				users[f] = true
			}
		}
	}
	return capStrings(keys(deps), limit), capStrings(keys(users), limit)
}

// body is the source of a symbol, from its line for a bounded number of lines.
func (ix *Index) body(sym Symbol) string {
	raw, err := os.ReadFile(filepath.Join(ix.Root, sym.File))
	if err != nil {
		return ""
	}
	lines := strings.Split(string(raw), "\n")
	start := sym.Line - 1
	if start < 0 || start >= len(lines) {
		return ""
	}
	end := min(start+rules.Ranking.BodyLines, len(lines))
	return strings.Join(lines[start:end], "\n")
}

func sortHits(hits []Hit) {
	sort.Slice(hits, func(i, j int) bool {
		if hits[i].Score != hits[j].Score {
			return hits[i].Score > hits[j].Score
		}
		return hits[i].Symbol.Name < hits[j].Symbol.Name
	})
}

func keys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func capStrings(s []string, limit int) []string {
	if limit > 0 && len(s) > limit {
		return s[:limit]
	}
	return s
}
