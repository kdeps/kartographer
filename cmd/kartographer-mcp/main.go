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
	flag.Parse()

	ix, err := codegraph.Build(*dir)
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
