package codegraph

import (
	"encoding/json"
	"fmt"

	"github.com/kdeps/kartographer/mcp"
)

// Default result limits, for the tools.
// Tools returns the code graph tools under stable names.
//
// strict_calls and code_only are accepted for compatibility. The index holds
// only resolved calls and only code nodes, so both have no effect yet.
func Tools(ix *Index) []mcp.Tool {
	tools := append(tierOneTools(ix), Tier2Tools(ix)...)
	return append(tools, Tier3Tools(ix)...)
}

func tierOneTools(ix *Index) []mcp.Tool {
	symbolArg := map[string]any{"type": "string", "description": "Symbol name, e.g. 'Type.Method' or 'pkg.Func'."}
	limitArg := func(def int) map[string]any {
		return map[string]any{"type": "integer", "minimum": 1, "maximum": rules.Limits.MaxResult, "default": def}
	}
	strictArg := map[string]any{"type": "boolean", "default": true, "description": "Accepted for compatibility; only resolved calls are indexed."}
	return []mcp.Tool{
		{
			Name:        "find_symbol",
			Description: "Locate where a function or method is defined. Matches a substring of the full symbol name.",
			InputSchema: schema(map[string]any{
				"term":      map[string]any{"type": "string", "description": "Substring of the symbol name."},
				"code_only": map[string]any{"type": "boolean", "default": false, "description": "Accepted for compatibility; every node is code."},
				"limit":     limitArg(rules.Limits.FindSymbol),
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
				return render(ix.Find(a.Term, pick(a.Limit, rules.Limits.FindSymbol)), nil)
			},
		},
		{
			Name:        "callers",
			Description: "List the functions that call a symbol directly.",
			InputSchema: schema(map[string]any{
				"symbol":       symbolArg,
				"strict_calls": strictArg,
				"limit":        limitArg(rules.Limits.Callers),
			}, "symbol"),
			Handle: func(raw json.RawMessage) (string, error) {
				var a struct {
					Symbol string `json:"symbol"`
					Limit  int    `json:"limit"`
				}
				if err := json.Unmarshal(raw, &a); err != nil {
					return "", err
				}
				found, err := ix.Callers(a.Symbol, pick(a.Limit, rules.Limits.Callers))
				if err != nil {
					return "", err
				}
				return render(found, nil)
			},
		},
		{
			Name:        "callees",
			Description: "List the functions a symbol calls directly.",
			InputSchema: schema(map[string]any{
				"symbol":       symbolArg,
				"strict_calls": strictArg,
				"limit":        limitArg(rules.Limits.Callees),
			}, "symbol"),
			Handle: func(raw json.RawMessage) (string, error) {
				var a struct {
					Symbol string `json:"symbol"`
					Limit  int    `json:"limit"`
				}
				if err := json.Unmarshal(raw, &a); err != nil {
					return "", err
				}
				found, err := ix.Callees(a.Symbol, pick(a.Limit, rules.Limits.Callees))
				if err != nil {
					return "", err
				}
				return render(found, nil)
			},
		},
		{
			Name:        "impact",
			Description: "Direct callers and callees of a symbol: a bounded first-pass scope for a change.",
			InputSchema: schema(map[string]any{
				"target":    symbolArg,
				"max_seeds": map[string]any{"type": "integer", "minimum": 1, "maximum": 30, "default": rules.Limits.ImpactSeeds},
				"limit":     limitArg(rules.Limits.Impact),
			}, "target"),
			Handle: func(raw json.RawMessage) (string, error) {
				var a struct {
					Target string `json:"target"`
					Limit  int    `json:"limit"`
				}
				if err := json.Unmarshal(raw, &a); err != nil {
					return "", err
				}
				found, err := ix.Neighbours(a.Target, pick(a.Limit, rules.Limits.Impact))
				if err != nil {
					return "", err
				}
				return render(found, nil)
			},
		},
		{
			Name:        "tests_for",
			Description: "Test functions that reach a symbol or a file through callers.",
			InputSchema: schema(map[string]any{
				"target": map[string]any{"type": "string", "description": "A symbol name, or a module-relative file path."},
				"limit":  limitArg(rules.Limits.TestsFor),
			}, "target"),
			Handle: func(raw json.RawMessage) (string, error) {
				var a struct {
					Target string `json:"target"`
					Limit  int    `json:"limit"`
				}
				if err := json.Unmarshal(raw, &a); err != nil {
					return "", err
				}
				found, err := ix.TestsFor(a.Target, pick(a.Limit, rules.Limits.TestsFor))
				if err != nil {
					return "", err
				}
				return render(found, nil)
			},
		},
		{
			Name:        "rank_files",
			Description: "Rank the files that match a question by the symbols in them. Keyword match, not semantic.",
			InputSchema: schema(map[string]any{
				"question": map[string]any{"type": "string", "description": "A question or a few keywords."},
				"limit":    limitArg(rules.Limits.RankFiles),
			}, "question"),
			Handle: func(raw json.RawMessage) (string, error) {
				var a struct {
					Question string `json:"question"`
					Limit    int    `json:"limit"`
				}
				if err := json.Unmarshal(raw, &a); err != nil {
					return "", err
				}
				if a.Question == "" {
					return "", fmt.Errorf("question is required")
				}
				hits := ix.RankFiles(a.Question, pick(a.Limit, rules.Limits.RankFiles))
				rows := make([]map[string]any, 0, len(hits))
				for _, h := range hits {
					rows = append(rows, map[string]any{"file": h.Symbol.File, "score": h.Score})
				}
				b, err := json.Marshal(rows)
				return string(b), err
			},
		},
		{
			Name:        "query",
			Description: "Find the symbols that match a question, with their source text cut to a budget. Keyword match, not semantic.",
			InputSchema: schema(map[string]any{
				"question": map[string]any{"type": "string", "description": "A question or a few keywords."},
				"budget":   map[string]any{"type": "integer", "minimum": 500, "maximum": 50000, "default": rules.Limits.QueryBudget, "description": "Approximate characters of source to return."},
				"k":        map[string]any{"type": "integer", "minimum": 1, "maximum": 100, "default": rules.Limits.QueryK},
				"skeleton": map[string]any{"type": "boolean", "default": false, "description": "Return locations only, without source."},
			}, "question"),
			Handle: func(raw json.RawMessage) (string, error) {
				var a struct {
					Question string `json:"question"`
					Budget   int    `json:"budget"`
					K        int    `json:"k"`
					Skeleton bool   `json:"skeleton"`
				}
				if err := json.Unmarshal(raw, &a); err != nil {
					return "", err
				}
				if a.Question == "" {
					return "", fmt.Errorf("question is required")
				}
				hits := ix.Query(a.Question, pick(a.K, rules.Limits.QueryK), max(a.Budget, 500), a.Skeleton)
				rows := make([]map[string]any, 0, len(hits))
				for _, h := range hits {
					row := map[string]any{"name": h.Symbol.Name, "file": h.Symbol.File, "line": h.Symbol.Line, "score": h.Score}
					if !a.Skeleton {
						row["body"] = h.Body
					}
					rows = append(rows, row)
				}
				b, err := json.Marshal(rows)
				return string(b), err
			},
		},
		{
			Name:        "file_neighbors",
			Description: "A file's dependencies (files it calls into) and dependents (files that call into it), from the call graph.",
			InputSchema: schema(map[string]any{
				"file":         map[string]any{"type": "string", "description": "Module-relative file path."},
				"limit":        limitArg(rules.Limits.FileNeighbors),
				"symbols_only": map[string]any{"type": "boolean", "default": false, "description": "Return the file's symbols only."},
			}, "file"),
			Handle: func(raw json.RawMessage) (string, error) {
				var a struct {
					File        string `json:"file"`
					Limit       int    `json:"limit"`
					SymbolsOnly bool   `json:"symbols_only"`
				}
				if err := json.Unmarshal(raw, &a); err != nil {
					return "", err
				}
				if a.File == "" {
					return "", fmt.Errorf("file is required")
				}
				if a.SymbolsOnly {
					return render(ix.FileSymbols(a.File), nil)
				}
				deps, users := ix.FileNeighbors(a.File, pick(a.Limit, rules.Limits.FileNeighbors))
				b, err := json.Marshal(map[string]any{"dependencies": deps, "dependents": users})
				return string(b), err
			},
		},
		{
			Name:        "stats",
			Description: "Counts of symbols and call edges in the indexed module.",
			InputSchema: schema(map[string]any{}),
			Handle: func(json.RawMessage) (string, error) {
				symbols, edges := ix.Stats()
				b, err := json.Marshal(map[string]any{"module": ix.Module, "symbols": symbols, "edges": edges})
				return string(b), err
			},
		},
		{
			Name:        "list_modules",
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
	if n < 1 || n > rules.Limits.MaxResult {
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
