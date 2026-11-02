package codegraph

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/kdeps/kartographer/mcp"
)

// Tier3Tools are the memory, workspace, risk and fetch tools. Notes are local (see package notes). Workspaces are one: this module.
// Options that only the hosted memory service has (profiles, sufficiency,
// grounding, time filters) are accepted and ignored. fetch takes a symbol name,
// not a repository:snapshot:node id, because this index has no snapshots.
func Tier3Tools(ix *Index) []mcp.Tool {
	return []mcp.Tool{
		{
			Name:        "remember",
			Description: "Save a durable note about the module: a decision, constraint, gotcha or rationale.",
			InputSchema: schema(map[string]any{
				"text": map[string]any{"type": "string", "description": "One self-contained statement."},
				"tags": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			}, "text"),
			Handle: func(raw json.RawMessage) (string, error) {
				var a struct {
					Text string   `json:"text"`
					Tags []string `json:"tags"`
				}
				if err := json.Unmarshal(raw, &a); err != nil {
					return "", err
				}
				n, err := ix.Notes.Add(a.Text, a.Tags)
				if err != nil {
					return "", err
				}
				b, err := json.Marshal(map[string]any{"saved": true, "id": n.ID})
				return string(b), err
			},
		},
		{
			Name:        "recall",
			Description: "Notes that share words with a query, best match first. Keyword match, not semantic.",
			InputSchema: schema(map[string]any{
				"query": map[string]any{"type": "string"},
				"k":     map[string]any{"type": "integer", "minimum": 1, "maximum": 100, "default": rules.Limits.Recall},
			}, "query"),
			Handle: func(raw json.RawMessage) (string, error) {
				var a struct {
					Query string `json:"query"`
					K     int    `json:"k"`
				}
				if err := json.Unmarshal(raw, &a); err != nil {
					return "", err
				}
				if a.Query == "" {
					return "", fmt.Errorf("query is required")
				}
				found, err := ix.Notes.Search(a.Query, pick(a.K, rules.Limits.Recall))
				if err != nil {
					return "", err
				}
				b, err := json.Marshal(map[string]any{"memories": found})
				return string(b), err
			},
		},
		{
			Name:        "memories_about",
			Description: "Notes that mention a file path or a symbol name.",
			InputSchema: schema(map[string]any{
				"target": map[string]any{"type": "string", "description": "A module-relative file path or a symbol name."},
				"k":      map[string]any{"type": "integer", "minimum": 1, "maximum": 20, "default": rules.Limits.MemoriesAbout},
			}, "target"),
			Handle: func(raw json.RawMessage) (string, error) {
				var a struct {
					Target string `json:"target"`
					K      int    `json:"k"`
				}
				if err := json.Unmarshal(raw, &a); err != nil {
					return "", err
				}
				if a.Target == "" {
					return "", fmt.Errorf("target is required")
				}
				found, err := ix.Notes.About(a.Target, pick(a.K, rules.Limits.MemoriesAbout))
				if err != nil {
					return "", err
				}
				b, err := json.Marshal(map[string]any{"memories": found})
				return string(b), err
			},
		},
		{
			Name:        "list_workspaces",
			Description: "The workspaces this server can target. It serves one: the module it indexed.",
			InputSchema: schema(map[string]any{}),
			Handle: func(json.RawMessage) (string, error) {
				b, err := json.Marshal(map[string]any{"active": ix.Module, "workspaces": []map[string]any{
					{"handle": ix.Module, "active": true},
				}})
				return string(b), err
			},
		},
		{
			Name:        "set_workspace",
			Description: "Select a workspace by handle. Only the indexed module is available.",
			InputSchema: schema(map[string]any{
				"handle":       map[string]any{"type": "string"},
				"make_default": map[string]any{"type": "boolean", "default": false},
			}, "handle"),
			Handle: func(raw json.RawMessage) (string, error) {
				var a struct {
					Handle string `json:"handle"`
				}
				if err := json.Unmarshal(raw, &a); err != nil {
					return "", err
				}
				if a.Handle != ix.Module {
					return "", fmt.Errorf("unknown workspace %q; the only one is %q", a.Handle, ix.Module)
				}
				b, err := json.Marshal(map[string]any{"active": ix.Module})
				return string(b), err
			},
		},
		{
			Name:        "impact_and_risk",
			Description: "Files ranked by how many symbols a change to the target touches, with the tests that reach the target. Static, not a verdict.",
			InputSchema: schema(map[string]any{
				"target": map[string]any{"type": "string", "description": "A symbol name."},
				"limit":  map[string]any{"type": "integer", "minimum": 1, "maximum": 300, "default": rules.Limits.ImpactFiles},
			}, "target"),
			Handle: func(raw json.RawMessage) (string, error) {
				var a struct {
					Target string `json:"target"`
					Limit  int    `json:"limit"`
				}
				if err := json.Unmarshal(raw, &a); err != nil {
					return "", err
				}
				files, tests, err := ix.ImpactRisk(a.Target, pick(a.Limit, rules.Limits.ImpactFiles))
				if err != nil {
					return "", err
				}
				b, err := json.Marshal(map[string]any{"files": files, "tests": names(tests)})
				return string(b), err
			},
		},
		{
			Name:        "fetch",
			Description: "The source of one symbol. The id is the symbol name.",
			InputSchema: schema(map[string]any{
				"id": map[string]any{"type": "string", "description": "A symbol name, as returned by search or the other tools."},
			}, "id"),
			Handle: func(raw json.RawMessage) (string, error) {
				var a struct {
					ID string `json:"id"`
				}
				if err := json.Unmarshal(raw, &a); err != nil {
					return "", err
				}
				sym, body, _, err := ix.Node(a.ID, 0)
				if err != nil {
					return "", err
				}
				if len(body) > rules.Limits.FetchBytes {
					body = body[:rules.Limits.FetchBytes]
				}
				b, err := json.Marshal(map[string]any{"name": sym.Name, "file": sym.File, "line": sym.Line, "body": body})
				return string(b), err
			},
		},
	}
}

// ImpactRisk ranks the files holding the target's direct callers and callees
// by how many of those symbols they hold, and lists the tests that reach the target.
func (ix *Index) ImpactRisk(target string, limit int) ([]map[string]any, []Symbol, error) {
	neigh, err := ix.Neighbours(target, 0)
	if err != nil {
		return nil, nil, err
	}
	perFile := map[string]int{}
	for _, s := range neigh {
		perFile[s.File]++
	}
	files := make([]map[string]any, 0, len(perFile))
	for f, n := range perFile {
		files = append(files, map[string]any{"file": f, "impacted": n})
	}
	sort.Slice(files, func(i, j int) bool {
		if files[i]["impacted"] != files[j]["impacted"] {
			return files[i]["impacted"].(int) > files[j]["impacted"].(int)
		}
		return files[i]["file"].(string) < files[j]["file"].(string)
	})
	if limit > 0 && len(files) > limit {
		files = files[:limit]
	}
	tests, err := ix.TestsFor(target, 0)
	if err != nil {
		return nil, nil, err
	}
	return files, tests, nil
}
