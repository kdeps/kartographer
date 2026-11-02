package codegraph

import (
	"encoding/json"
	"fmt"

	"github.com/kdeps/kartographer/mcp"
)

// Tier2Tools are the path, neighbourhood and source tools.
// references and imports_exports are not served: they need
// non-call references and import statements, which the index does not store.
func Tier2Tools(ix *Index) []mcp.Tool {
	symbolArg := map[string]any{"type": "string", "description": "Symbol name, e.g. 'Type.Method' or 'pkg.Func'."}
	return []mcp.Tool{
		{
			Name:        "node",
			Description: "A symbol's source plus its direct callers and callees.",
			InputSchema: schema(map[string]any{
				"symbol":       symbolArg,
				"strict_calls": map[string]any{"type": "boolean", "default": false, "description": "Accepted for compatibility; only resolved calls are indexed."},
				"limit":        map[string]any{"type": "integer", "minimum": 1, "maximum": 100, "default": rules.Limits.NodeNeighbours},
			}, "symbol"),
			Handle: func(raw json.RawMessage) (string, error) {
				var a struct {
					Symbol string `json:"symbol"`
					Limit  int    `json:"limit"`
				}
				if err := json.Unmarshal(raw, &a); err != nil {
					return "", err
				}
				sym, body, neigh, err := ix.Node(a.Symbol, pick(a.Limit, rules.Limits.NodeNeighbours))
				if err != nil {
					return "", err
				}
				b, err := json.Marshal(map[string]any{
					"name": sym.Name, "file": sym.File, "line": sym.Line, "body": body,
					"neighbours": neigh,
				})
				return string(b), err
			},
		},
		{
			Name:        "trace",
			Description: "Directed call paths from one symbol to another, shortest first.",
			InputSchema: schema(map[string]any{
				"src":          map[string]any{"type": "string", "description": "Symbol to trace from."},
				"tgt":          map[string]any{"type": "string", "description": "Symbol to trace to."},
				"strict_calls": map[string]any{"type": "boolean", "default": true, "description": "Accepted for compatibility; only resolved calls are indexed."},
				"k_paths":      map[string]any{"type": "integer", "minimum": 1, "maximum": 10, "default": 3},
			}, "src", "tgt"),
			Handle: func(raw json.RawMessage) (string, error) {
				var a struct {
					Src    string `json:"src"`
					Tgt    string `json:"tgt"`
					KPaths int    `json:"k_paths"`
				}
				if err := json.Unmarshal(raw, &a); err != nil {
					return "", err
				}
				paths, err := ix.Trace(a.Src, a.Tgt, pick(a.KPaths, 3))
				if err != nil {
					return "", err
				}
				b, err := json.Marshal(map[string]any{"paths": paths})
				return string(b), err
			},
		},
		{
			Name:        "shortest_path",
			Description: "The shortest connection between two symbols, following calls in either direction.",
			InputSchema: schema(map[string]any{
				"src": map[string]any{"type": "string", "description": "Symbol to start from."},
				"tgt": map[string]any{"type": "string", "description": "Symbol to reach."},
			}, "src", "tgt"),
			Handle: func(raw json.RawMessage) (string, error) {
				var a struct {
					Src string `json:"src"`
					Tgt string `json:"tgt"`
				}
				if err := json.Unmarshal(raw, &a); err != nil {
					return "", err
				}
				path, err := ix.ShortestPath(a.Src, a.Tgt)
				if err != nil {
					return "", err
				}
				b, err := json.Marshal(map[string]any{"path": path, "connected": path != nil})
				return string(b), err
			},
		},
		{
			Name:        "expand",
			Description: "Source of each named symbol. Names are the symbol names returned by other tools.",
			InputSchema: schema(map[string]any{
				"ids": map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "Symbol names to read."},
			}, "ids"),
			Handle: func(raw json.RawMessage) (string, error) {
				var a struct {
					IDs []string `json:"ids"`
				}
				if err := json.Unmarshal(raw, &a); err != nil {
					return "", err
				}
				if len(a.IDs) == 0 {
					return "", fmt.Errorf("ids is required")
				}
				b, err := json.Marshal(ix.Expand(a.IDs))
				return string(b), err
			},
		},
		{
			Name:        "find_seeds",
			Description: "The top symbols for a question, without their source. Keyword match, not semantic.",
			InputSchema: schema(map[string]any{
				"question": map[string]any{"type": "string"},
				"limit":    map[string]any{"type": "integer", "minimum": 1, "maximum": 100, "default": rules.Limits.FindSeeds},
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
				return render(symbolsOf(ix.Query(a.Question, pick(a.Limit, rules.Limits.FindSeeds), 0, true)), nil)
			},
		},
		{
			Name:        "search",
			Description: "Symbols whose name contains a substring.",
			InputSchema: schema(map[string]any{
				"query": map[string]any{"type": "string", "description": "A fragment of a symbol name."},
			}, "query"),
			Handle: func(raw json.RawMessage) (string, error) {
				var a struct {
					Query string `json:"query"`
				}
				if err := json.Unmarshal(raw, &a); err != nil {
					return "", err
				}
				if a.Query == "" {
					return "", fmt.Errorf("query is required")
				}
				return render(ix.Find(a.Query, rules.Limits.MaxResult), nil)
			},
		},
		{
			Name:        "render_subgraph",
			Description: "The nodes and call edges around seed symbols, for drawing a graph.",
			InputSchema: schema(map[string]any{
				"seeds":      map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
				"depth":      map[string]any{"type": "integer", "minimum": 1, "maximum": 3, "default": rules.Limits.RenderDepth},
				"node_limit": map[string]any{"type": "integer", "minimum": 1, "maximum": 2000, "default": rules.Limits.RenderNodes},
				"edge_limit": map[string]any{"type": "integer", "minimum": 1, "maximum": 20000, "default": rules.Limits.RenderEdges},
			}, "seeds"),
			Handle: func(raw json.RawMessage) (string, error) {
				var a struct {
					Seeds     []string `json:"seeds"`
					Depth     int      `json:"depth"`
					NodeLimit int      `json:"node_limit"`
					EdgeLimit int      `json:"edge_limit"`
				}
				if err := json.Unmarshal(raw, &a); err != nil {
					return "", err
				}
				if len(a.Seeds) == 0 {
					return "", fmt.Errorf("seeds is required")
				}
				nodes, edges := ix.Neighbourhood(a.Seeds, pick(a.Depth, rules.Limits.RenderDepth), pick(a.NodeLimit, rules.Limits.RenderNodes))
				if limit := pick(a.EdgeLimit, rules.Limits.RenderEdges); len(edges) > limit {
					edges = edges[:limit]
				}
				links := make([]map[string]string, 0, len(edges))
				for _, e := range edges {
					links = append(links, map[string]string{"source": e[0], "target": e[1]})
				}
				b, err := json.Marshal(map[string]any{"nodes": nodes, "links": links})
				return string(b), err
			},
		},
	}
}

func symbolsOf(hits []Hit) []Symbol {
	out := make([]Symbol, 0, len(hits))
	for _, h := range hits {
		out = append(out, h.Symbol)
	}
	return out
}
