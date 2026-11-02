package codegraph

import (
	"encoding/json"
	"fmt"

	"github.com/kdeps/kartographer/mcp"
)

// RefTools are the reference and import tools. references covers Go only as
// far as the SSA form records non-call uses; imports_exports reads Go files.
func RefTools(ix *Index) []mcp.Tool {
	return []mcp.Tool{
		{
			Name:        "references",
			Description: "Non-call uses of a symbol: where it is passed, stored or captured as a value (Go).",
			InputSchema: schema(map[string]any{
				"symbol": map[string]any{"type": "string", "description": "Symbol name."},
				"limit":  map[string]any{"type": "integer", "minimum": 1, "maximum": rules.Limits.MaxResult, "default": rules.Limits.References},
			}, "symbol"),
			Handle: func(raw json.RawMessage) (string, error) {
				var a struct {
					Symbol string `json:"symbol"`
					Limit  int    `json:"limit"`
				}
				if err := json.Unmarshal(raw, &a); err != nil {
					return "", err
				}
				refs, err := ix.References(a.Symbol, pick(a.Limit, rules.Limits.References))
				if err != nil {
					return "", err
				}
				b, err := json.Marshal(map[string]any{"references": refs})
				return string(b), err
			},
		},
		{
			Name:        "imports_exports",
			Description: "The import paths and exported names of a Go file.",
			InputSchema: schema(map[string]any{
				"file":  map[string]any{"type": "string", "description": "Module-relative Go file path."},
				"limit": map[string]any{"type": "integer", "minimum": 1, "maximum": rules.Limits.MaxResult, "default": rules.Limits.ImportsExports},
			}, "file"),
			Handle: func(raw json.RawMessage) (string, error) {
				var a struct {
					File  string `json:"file"`
					Limit int    `json:"limit"`
				}
				if err := json.Unmarshal(raw, &a); err != nil {
					return "", err
				}
				if a.File == "" {
					return "", fmt.Errorf("file is required")
				}
				imports, exports, err := ix.ImportsExports(a.File)
				if err != nil {
					return "", err
				}
				limit := pick(a.Limit, rules.Limits.ImportsExports)
				b, err := json.Marshal(map[string]any{
					"imports": capStrings(imports, limit),
					"exports": capStrings(exports, limit),
				})
				return string(b), err
			},
		},
	}
}
