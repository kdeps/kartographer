package codegraph

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/kdeps/kartographer/mcp"
)

// ChangedLines reads a unified diff and returns, per file, the line ranges it
// touches on the new side, 1-based and inclusive. A hunk that only removes
// lines marks the line the removal was at. Files that were deleted are left out.
// Paths are as the diff names them, so run git with --relative for a module
// that is not at the repository root.
func ChangedLines(diff string) map[string][][2]int {
	out := map[string][][2]int{}
	file := ""
	for _, line := range strings.Split(diff, "\n") {
		switch {
		case strings.HasPrefix(line, "+++ "):
			file = strings.TrimPrefix(strings.TrimSpace(line[len("+++ "):]), "b/")
		case strings.HasPrefix(line, "@@") && file != "" && file != "/dev/null":
			start, count, ok := newSide(line)
			if !ok {
				continue
			}
			if count == 0 {
				start = max(start, 1)
				out[file] = append(out[file], [2]int{start, start})
				continue
			}
			out[file] = append(out[file], [2]int{start, start + count - 1})
		}
	}
	return out
}

// newSide reads the new-side start and count of a hunk header such as
// "@@ -3,2 +5,4 @@". A count left out is 1.
func newSide(header string) (start, count int, ok bool) {
	for _, f := range strings.Fields(header) {
		if !strings.HasPrefix(f, "+") {
			continue
		}
		parts := strings.SplitN(strings.TrimPrefix(f, "+"), ",", 2)
		s, err := strconv.Atoi(parts[0])
		if err != nil {
			return 0, 0, false
		}
		count = 1
		if len(parts) == 2 {
			if count, err = strconv.Atoi(parts[1]); err != nil {
				return 0, 0, false
			}
		}
		return s, count, true
	}
	return 0, 0, false
}

// Changed returns the symbols whose declarations a diff touches, in file and
// line order.
func (ix *Index) Changed(diff string) []Symbol {
	ranges := ChangedLines(diff)
	var out []Symbol
	for _, s := range ix.Symbols {
		end := max(s.End, s.Line)
		for _, r := range ranges[s.File] {
			if r[0] <= end && r[1] >= s.Line {
				out = append(out, s)
				break
			}
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].File != out[j].File {
			return out[i].File < out[j].File
		}
		return out[i].Line < out[j].Line
	})
	return out
}

// Impact is what a diff can affect: the symbols it touches, then the symbols
// that depend on them up to depth call hops away. limit caps the result.
func (ix *Index) Impact(diff string, depth, limit int) []Symbol {
	seen := map[string]bool{}
	var out []Symbol
	add := func(s Symbol) {
		if !seen[s.Name] {
			seen[s.Name] = true
			out = append(out, s)
		}
	}
	changed := ix.Changed(diff)
	for _, s := range changed {
		add(s)
	}
	for _, s := range changed {
		if dependents, err := ix.Affected(s.Name, depth, limit); err == nil {
			for _, d := range dependents {
				add(d)
			}
		}
	}
	return capped(out, limit)
}

// ChangesTools offers Impact to an agent, given the unified diff text.
func ChangesTools(ix *Index) []mcp.Tool {
	return []mcp.Tool{{
		Name:        "impact_of_diff",
		Description: "The symbols a unified diff touches, and the symbols that depend on them. Pass the diff text, e.g. from git diff --unified=0.",
		InputSchema: schema(map[string]any{
			"diff":  map[string]any{"type": "string", "description": "A unified diff."},
			"depth": map[string]any{"type": "integer", "minimum": 0, "maximum": 10, "default": 2, "description": "Caller hops to follow from each changed symbol."},
			"limit": limitArg(60),
		}, "diff"),
		Handle: func(raw json.RawMessage) (string, error) {
			var a struct {
				Diff  string `json:"diff"`
				Depth int    `json:"depth"`
				Limit int    `json:"limit"`
			}
			if err := json.Unmarshal(raw, &a); err != nil {
				return "", err
			}
			if a.Diff == "" {
				return "", fmt.Errorf("diff is required")
			}
			depth := a.Depth
			if depth <= 0 {
				depth = 2
			}
			return render(ix.Impact(a.Diff, depth, pick(a.Limit, 60)), nil)
		},
	}}
}

func limitArg(def int) map[string]any {
	return map[string]any{"type": "integer", "minimum": 1, "maximum": 300, "default": def}
}
