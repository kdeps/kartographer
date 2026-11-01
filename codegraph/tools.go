package codegraph

import (
	"encoding/json"
	"fmt"

	"github.com/kdeps/kartographer/mcp"
)

// Default result limits, matching the Graphify tools.
const (
	findLimit      = 30
	callerLimit    = 25
	calleeLimit    = 25
	impactLimit    = 60
	impactSeeds    = 8
	testsForLimit  = 25
	maxResultLimit = 300
)

// Tools returns the code graph tools under the names Graphify uses, so a
// client that calls Graphify can call this server unchanged.
//
// strict_calls and code_only are accepted for compatibility. The index holds
// only resolved calls and only code nodes, so both have no effect yet.
func Tools(ix *Index) []mcp.Tool {
	symbolArg := map[string]any{"type": "string", "description": "Symbol name, e.g. 'Type.Method' or 'pkg.Func'."}
	limitArg := func(def int) map[string]any {
		return map[string]any{"type": "integer", "minimum": 1, "maximum": maxResultLimit, "default": def}
	}
	strictArg := map[string]any{"type": "boolean", "default": true, "description": "Accepted for compatibility; only resolved calls are indexed."}
	return []mcp.Tool{
		{
			Name:        "graphify_find",
			Description: "Locate where a function or method is defined. Matches a substring of the full symbol name.",
			InputSchema: schema(map[string]any{
				"term":      map[string]any{"type": "string", "description": "Substring of the symbol name."},
				"code_only": map[string]any{"type": "boolean", "default": false, "description": "Accepted for compatibility; every node is code."},
				"limit":     limitArg(findLimit),
			}, "term"),
			Handle: func(raw json.RawMessage) (string, error) {
				var a struct {
					Term  string `json:"term"`
					Limit int    `json:"limit"`
				}
				if err := json.Unmarshal(raw, &a); err != nil {
					return "", err
				}
				if a.Term == "" {
					return "", fmt.Errorf("term is required")
				}
				return render(ix.Find(a.Term, pick(a.Limit, findLimit)), nil)
			},
		},
		{
			Name:        "graphify_callers",
			Description: "List the functions that call a symbol directly.",
			InputSchema: schema(map[string]any{
				"symbol":       symbolArg,
				"strict_calls": strictArg,
				"limit":        limitArg(callerLimit),
			}, "symbol"),
			Handle: func(raw json.RawMessage) (string, error) {
				var a struct {
					Symbol string `json:"symbol"`
					Limit  int    `json:"limit"`
				}
				if err := json.Unmarshal(raw, &a); err != nil {
					return "", err
				}
				found, err := ix.Callers(a.Symbol, pick(a.Limit, callerLimit))
				if err != nil {
					return "", err
				}
				return render(found, nil)
			},
		},
		{
			Name:        "graphify_callees",
			Description: "List the functions a symbol calls directly.",
			InputSchema: schema(map[string]any{
				"symbol":       symbolArg,
				"strict_calls": strictArg,
				"limit":        limitArg(calleeLimit),
			}, "symbol"),
			Handle: func(raw json.RawMessage) (string, error) {
				var a struct {
					Symbol string `json:"symbol"`
					Limit  int    `json:"limit"`
				}
				if err := json.Unmarshal(raw, &a); err != nil {
					return "", err
				}
				found, err := ix.Callees(a.Symbol, pick(a.Limit, calleeLimit))
				if err != nil {
					return "", err
				}
				return render(found, nil)
			},
		},
		{
			Name:        "graphify_impact",
			Description: "Direct callers and callees of a symbol: a bounded first-pass scope for a change.",
			InputSchema: schema(map[string]any{
				"target":    symbolArg,
				"max_seeds": map[string]any{"type": "integer", "minimum": 1, "maximum": 30, "default": impactSeeds},
				"limit":     limitArg(impactLimit),
			}, "target"),
			Handle: func(raw json.RawMessage) (string, error) {
				var a struct {
					Target string `json:"target"`
					Limit  int    `json:"limit"`
				}
				if err := json.Unmarshal(raw, &a); err != nil {
					return "", err
				}
				found, err := ix.Neighbours(a.Target, pick(a.Limit, impactLimit))
				if err != nil {
					return "", err
				}
				return render(found, nil)
			},
		},
		{
			Name:        "graphify_tests_for",
			Description: "Test functions that reach a symbol or a file through callers.",
			InputSchema: schema(map[string]any{
				"target": map[string]any{"type": "string", "description": "A symbol name, or a module-relative file path."},
				"limit":  limitArg(testsForLimit),
			}, "target"),
			Handle: func(raw json.RawMessage) (string, error) {
				var a struct {
					Target string `json:"target"`
					Limit  int    `json:"limit"`
				}
				if err := json.Unmarshal(raw, &a); err != nil {
					return "", err
				}
				found, err := ix.TestsFor(a.Target, pick(a.Limit, testsForLimit))
				if err != nil {
					return "", err
				}
				return render(found, nil)
			},
		},
		{
			Name:        "graph_stats",
			Description: "Counts of symbols and call edges in the indexed module.",
			InputSchema: schema(map[string]any{}),
			Handle: func(json.RawMessage) (string, error) {
				symbols, edges := ix.Stats()
				b, err := json.Marshal(map[string]any{"module": ix.Module, "symbols": symbols, "edges": edges})
				return string(b), err
			},
		},
		{
			Name:        "list_repositories",
			Description: "The module this server indexed. Its full_name is the repository_id.",
			InputSchema: schema(map[string]any{}),
			Handle: func(json.RawMessage) (string, error) {
				b, err := json.Marshal(map[string]any{"repositories": []map[string]any{
					{"full_name": ix.Module, "repository_id": ix.Module, "queryable": true},
				}})
				return string(b), err
			},
		},
	}
}

func schema(props map[string]any, required ...string) map[string]any {
	s := map[string]any{"type": "object", "properties": props}
	if len(required) > 0 {
		s["required"] = required
	}
	return s
}

// pick returns n when it is a valid limit, else def.
func pick(n, def int) int {
	if n < 1 || n > maxResultLimit {
		return def
	}
	return n
}

// render returns symbols as a JSON list of name, file and line.
func render(found []Symbol, err error) (string, error) {
	if err != nil {
		return "", err
	}
	type row struct {
		Name   string `json:"name"`
		File   string `json:"file"`
		Line   int    `json:"line"`
		IsTest bool   `json:"is_test,omitempty"`
	}
	rows := make([]row, 0, len(found))
	for _, s := range found {
		rows = append(rows, row{s.Name, s.File, s.Line, s.IsTest})
	}
	b, err := json.Marshal(rows)
	return string(b), err
}
