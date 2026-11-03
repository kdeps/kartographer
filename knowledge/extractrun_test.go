package knowledge

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
)

func TestRunExtraction(t *testing.T) {
	var calls atomic.Int32
	ask := func(_ context.Context, piece string) (string, error) {
		calls.Add(1)
		switch {
		case strings.Contains(piece, "down"):
			return "", errors.New("model down")
		case strings.Contains(piece, "junk"):
			return "no json", nil
		}
		return `{"entities":[{"name":"Stripe","kind":"tool"},{"name":"stripe","kind":"tool"},{"name":"Plans","kind":"tool"}]}`, nil
	}
	docs := []ExtractDoc{
		{Key: "a.md", Hash: "h1", Pieces: []string{"one", "two"}},
		{Key: "b.md", Hash: "h2", Pieces: []string{"down"}},
		{Key: "c.md", Hash: "h3", Pieces: []string{"junk"}},
		{Key: "d.md", Hash: "h4", Err: errors.New("bad header")},
	}
	cache := map[string]Extraction{}
	done, failed := RunExtraction(t.Context(), docs, cache, RunOptions{
		Model: "m", Workers: 2, MaxEntities: 1, Parse: testParse, Ask: ask})
	if done != 1 || failed != 3 {
		t.Fatalf("done=%d failed=%d", done, failed)
	}
	if a := cache["a.md"]; len(a.Entities) != 1 || a.Model != "m" || a.At.IsZero() || a.Error != "" {
		t.Errorf("entities are capped and deduplicated: %+v", a)
	}
	if !strings.Contains(cache["c.md"].Error, "unusable reply") || cache["b.md"].Error != "model down" ||
		cache["d.md"].Error != "bad header" {
		t.Errorf("errors kept: %+v", cache)
	}
	if NeedsExtraction(cache, "a.md", "h1") || !NeedsExtraction(cache, "a.md", "new") ||
		!NeedsExtraction(cache, "b.md", "h2") || !NeedsExtraction(cache, "zz.md", "h") {
		t.Error("NeedsExtraction wrong")
	}
	PruneCache(cache, func(k string) bool { return k == "a.md" })
	if len(cache) != 1 {
		t.Errorf("prune kept %v", cache)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	before := calls.Load()
	if d, f := RunExtraction(ctx, docs, map[string]Extraction{}, RunOptions{Ask: ask, Parse: testParse}); d+f != 0 ||
		calls.Load() != before {
		t.Error("a done context starts nothing")
	}
}
