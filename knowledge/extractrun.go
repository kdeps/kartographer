package knowledge

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

// Extraction is what a model found in one version of a document: the entry an
// extraction cache keeps per document key.
type Extraction struct {
	Hash      string     `json:"hash"`
	Model     string     `json:"model,omitempty"`
	At        time.Time  `json:"at,omitempty"`
	Entities  []Entity   `json:"entities"`
	Relations []Relation `json:"relations"`
	Error     string     `json:"error,omitempty"`
}

// ExtractDoc is a document to extract: its cache key, its version hash, and
// the pieces the model reads (or the error that stopped them being made).
type ExtractDoc struct {
	Key    string
	Hash   string
	Pieces []string
	Err    error
}

// RunOptions are how an extraction run asks the model and keeps its answers.
// The caller owns them.
type RunOptions struct {
	Model       string // recorded with each result
	Workers     int    // documents asked about at once, after the first; below 1 means 1
	MaxEntities int    // entities kept per document, the first unique names
	Parse       ParseOptions
	Ask         func(ctx context.Context, piece string) (string, error)
}

// NeedsExtraction reports whether cache has no successful extraction of the
// document key at version hash.
func NeedsExtraction(cache map[string]Extraction, key, hash string) bool {
	e, ok := cache[key]
	return !ok || e.Hash != hash || e.Error != ""
}

// RunExtraction asks the model about each document and stores the result in
// cache under its key. The first document runs alone, since a local model
// loads (or starts) on its first call; the rest run Workers at a time. It
// stops starting documents once ctx is done, and returns how many succeeded
// and how many failed.
func RunExtraction(ctx context.Context, docs []ExtractDoc, cache map[string]Extraction, o RunOptions) (done, failed int) {
	workers := max(o.Workers, 1)
	var mu sync.Mutex
	run := func(d ExtractDoc) {
		ex := extractDoc(ctx, d, o)
		mu.Lock()
		defer mu.Unlock()
		cache[d.Key] = ex
		if ex.Error != "" {
			failed++
		} else {
			done++
		}
	}
	if len(docs) > 0 && ctx.Err() == nil {
		run(docs[0])
		docs = docs[1:]
	}
	var wg sync.WaitGroup
	sem := make(chan struct{}, workers)
	for _, d := range docs {
		if ctx.Err() != nil {
			break
		}
		wg.Go(func() {
			sem <- struct{}{}
			defer func() { <-sem }()
			run(d)
		})
	}
	wg.Wait()
	return done, failed
}

// extractDoc asks about each piece of d and merges the validated answers.
func extractDoc(ctx context.Context, d ExtractDoc, o RunOptions) Extraction {
	ex := Extraction{Hash: d.Hash, Model: o.Model}
	if d.Err != nil {
		ex.Error = d.Err.Error()
		return ex
	}
	seen := map[string]bool{}
	for _, piece := range d.Pieces {
		reply, err := o.Ask(ctx, piece)
		if err != nil {
			ex.Error = err.Error()
			return ex
		}
		part, err := ParseFound(reply, o.Parse)
		if err != nil {
			ex.Error = fmt.Sprintf("unusable reply (%v): %.200q", err, reply)
			return ex
		}
		for _, e := range part.Entities {
			if key := strings.ToLower(e.Name); !seen[key] && len(ex.Entities) < o.MaxEntities {
				seen[key] = true
				ex.Entities = append(ex.Entities, e)
			}
		}
		ex.Relations = append(ex.Relations, part.Relations...)
	}
	ex.At = time.Now().UTC()
	return ex
}

// PruneCache removes the cache entries whose key keep rejects, such as those
// of documents that no longer exist.
func PruneCache(cache map[string]Extraction, keep func(key string) bool) {
	for k := range cache {
		if !keep(k) {
			delete(cache, k)
		}
	}
}
