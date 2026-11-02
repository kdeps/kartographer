// Command kartographer-mcp serves the code graph of a Go module over MCP on
// stdio, with the tool names.
//
//	kartographer-mcp -dir /path/to/module
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/kdeps/kartographer/codegraph"
	"github.com/kdeps/kartographer/mcp"
)

const version = "0.3.0"

func main() {
	dir := flag.String("dir", ".", "module root to index")
	noCache := flag.Bool("no-cache", false, "rebuild the index instead of reusing the saved one")
	flag.Parse()

	ix, err := load(*dir, *noCache)
	if err != nil {
		fmt.Fprintln(os.Stderr, "kartographer-mcp:", err)
		os.Exit(1)
	}
	symbols, edges := ix.Stats()
	fmt.Fprintf(os.Stderr, "kartographer-mcp: indexed %s: %d symbols, %d call edges\n", ix.Module, symbols, edges)

	srv := &mcp.Server{Name: "kartographer", Version: version, Tools: codegraph.Tools(ix)}
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
