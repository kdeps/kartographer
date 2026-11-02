// Command kartographer-mcp serves the code graph of a Go module over MCP on
// stdio, with the tool names.
//
//	kartographer-mcp -dir /path/to/module
package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/kdeps/kartographer/codegraph"
	"github.com/kdeps/kartographer/mcp"
)

const version = "0.3.0"

func main() {
	dir := flag.String("dir", ".", "module root to index")
	noCache := flag.Bool("no-cache", false, "rebuild the index instead of reusing the saved one")
	listen := flag.String("listen", "", "serve MCP over HTTP at this address (POST /) instead of stdio, e.g. :8080")
	rulesPath := flag.String("rules", "", "YAML file of business rules (ranking, limits, stopwords); kdeps ships one")
	flag.Parse()

	if *rulesPath == "" {
		fmt.Fprintln(os.Stderr, "kartographer-mcp: -rules is required")
		os.Exit(2)
	}
	raw, err := os.ReadFile(*rulesPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "kartographer-mcp:", err)
		os.Exit(1)
	}
	if err := codegraph.Configure(raw); err != nil {
		fmt.Fprintln(os.Stderr, "kartographer-mcp: rules:", err)
		os.Exit(1)
	}

	ix, err := load(*dir, *noCache)
	if err != nil {
		fmt.Fprintln(os.Stderr, "kartographer-mcp:", err)
		os.Exit(1)
	}
	symbols, edges := ix.Stats()
	fmt.Fprintf(os.Stderr, "kartographer-mcp: indexed %s: %d symbols, %d call edges\n", ix.Module, symbols, edges)

	srv := &mcp.Server{Name: "kartographer", Version: version, Tools: codegraph.Tools(ix)}
	if *listen != "" {
		hs := &http.Server{Addr: *listen, Handler: srv.Handler(), ReadHeaderTimeout: 10 * time.Second}
		fmt.Fprintf(os.Stderr, "kartographer-mcp: serving on %s\n", *listen)
		if err := hs.ListenAndServe(); err != nil {
			fmt.Fprintln(os.Stderr, "kartographer-mcp:", err)
			os.Exit(1)
		}
		return
	}
	if err := srv.Serve(os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "kartographer-mcp:", err)
		os.Exit(1)
	}
}

// load returns the index of dir, from the saved copy unless noCache is set.
func load(dir string, noCache bool) (*codegraph.Index, error) {
	if noCache {
		return codegraph.Build(dir)
	}
	cache, err := codegraph.CacheDir()
	if err != nil {
		return nil, err
	}
	return codegraph.BuildCached(dir, cache)
}
