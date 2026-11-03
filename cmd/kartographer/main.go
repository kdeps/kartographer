// Command kartographer queries the code graph of a module from the command
// line, with the same commands as Graphify's CLI: update, query, path, explain
// and affected.
//
//	kartographer -rules rules.yaml update [dir]
//	kartographer -rules rules.yaml query "<question>" [--dir D] [--k N] [--budget N] [--skeleton]
//	kartographer -rules rules.yaml path "<source>" "<target>" [--dir D] [--directed]
//	kartographer -rules rules.yaml explain "<symbol>" [--dir D]
//	kartographer -rules rules.yaml affected "<symbol>" [--dir D] [--depth N] [--limit N]
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/kdeps/kartographer/codegraph"
)

const usage = `usage: kartographer -rules FILE <command> [flags] [args]

commands:
  update [dir]                       build or refresh the code index
  query "<question>"                 find the symbols that match a question
  path "<source>" "<target>"         the connection between two symbols
  explain "<symbol>"                 a symbol with its source and neighbours
  affected "<symbol>"                symbols that depend on it, by distance
  watch [dir]                        rebuild the index when files change
  hook install|uninstall|status      rebuild after commits and checkouts (set KARTOGRAPHER_RULES)
  export FORMAT [-o FILE|DIR]        write the graph: graphml, html, svg, obsidian, wiki, neo4j, falkordb
  changes [base]                     the symbols that changes against base (default HEAD) touch, and what they affect
  add "<url>" [--dir-raw raw]        fetch a page into the raw directory (--author, --contributor)

flags: -rules FILE (required), -dir DIR (default .), -no-cache`

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run executes one command and returns the process exit code.
func run(args []string, out, errw io.Writer) int {
	global := flag.NewFlagSet("kartographer", flag.ContinueOnError)
	global.SetOutput(errw)
	rulesPath := global.String("rules", "", "YAML file of business rules")
	if err := global.Parse(args); err != nil {
		return 2
	}
	rest := global.Args()
	if len(rest) == 0 {
		_, _ = fmt.Fprintln(errw, usage)
		return 2
	}
	if *rulesPath == "" {
		_, _ = fmt.Fprintln(errw, "kartographer: -rules is required")
		return 2
	}
	raw, err := os.ReadFile(*rulesPath)
	if err == nil {
		err = codegraph.Configure(raw)
	}
	if err != nil {
		_, _ = fmt.Fprintln(errw, "kartographer: rules:", err)
		return 1
	}
	if err := execute(rest[0], rest[1:], out); err != nil {
		_, _ = fmt.Fprintln(errw, "kartographer:", err)
		if errors.Is(err, errUsage) {
			return 2
		}
		return 1
	}
	return 0
}

var errUsage = errors.New("bad usage")

//nolint:gochecknoglobals // fixed command list
var knownCommands = map[string]bool{"update": true, "query": true, "path": true, "explain": true, "affected": true,
	"watch": true, "hook": true, "export": true, "add": true, "changes": true}

// execute parses one command's flags and arguments, then runs it.
func execute(cmd string, args []string, out io.Writer) error {
	fs := flag.NewFlagSet(cmd, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	dir := fs.String("dir", ".", "module root")
	noCache := fs.Bool("no-cache", false, "rebuild instead of reusing the saved index")
	k := fs.Int("k", 5, "query: symbols to return")
	budget := fs.Int("budget", 5000, "query: characters of source to return")
	skeleton := fs.Bool("skeleton", false, "query: locations only, no source")
	directed := fs.Bool("directed", false, "path: follow calls one way only")
	depth := fs.Int("depth", 2, "affected: how many caller hops to follow")
	limit := fs.Int("limit", 60, "affected: symbols to return")
	interval := fs.Duration("interval", 2*time.Second, "watch: how often to check for changes")
	outPath := fs.String("o", "", "export: file or directory to write (default stdout)")
	rawDir := fs.String("dir-raw", "raw", "add: directory for the fetched page")
	author := fs.String("author", "", "add: author recorded in the front matter")
	contributor := fs.String("contributor", "", "add: contributor recorded in the front matter")
	positional, flagArgs := splitArgs(args)
	if err := fs.Parse(flagArgs); err != nil {
		return fmt.Errorf("%w: %v", errUsage, err)
	}
	positional = append(positional, fs.Args()...)

	if cmd == "update" {
		target := *dir
		if len(positional) > 0 {
			target = positional[0]
		}
		ix, err := load(target, *noCache)
		if err != nil {
			return err
		}
		symbols, edges := ix.Stats()
		_, _ = fmt.Fprintf(out, "indexed %s: %d symbols, %d call edges\n", ix.Module, symbols, edges)
		for _, m := range ix.Missing {
			_, _ = fmt.Fprintf(out, "missing: %s\n", m)
		}
		return nil
	}

	if !knownCommands[cmd] {
		return fmt.Errorf("%w: unknown command %q", errUsage, cmd)
	}
	switch cmd {
	case "watch":
		target := *dir
		if len(positional) > 0 {
			target = positional[0]
		}
		return watchCommand(out, target, *interval, 0, *noCache)
	case "hook":
		if len(positional) == 0 {
			return fmt.Errorf("%w: hook needs install, uninstall or status", errUsage)
		}
		return hookCommand(out, *dir, positional[0])
	case "changes":
		base := "HEAD"
		if len(positional) > 0 {
			base = positional[0]
		}
		return changesCommand(out, *dir, base, *depth, *limit, *noCache)
	case "add":
		if len(positional) == 0 {
			return fmt.Errorf("%w: add needs a URL", errUsage)
		}
		return addCommand(out, positional[0], *rawDir, *author, *contributor)
	case "export":
		if len(positional) == 0 {
			return fmt.Errorf("%w: export needs a format (graphml, html, svg, obsidian, wiki, neo4j, falkordb)", errUsage)
		}
		return exportCommand(out, positional[0], *outPath, *dir, *noCache)
	}
	if len(positional) == 0 || (cmd == "path" && len(positional) < 2) {
		return fmt.Errorf("%w: %s needs %s", errUsage, cmd, map[string]string{
			"query": `"<question>"`, "path": `"<source>" "<target>"`,
			"explain": `"<symbol>"`, "affected": `"<symbol>"`,
		}[cmd])
	}
	ix, err := load(*dir, *noCache)
	if err != nil {
		return err
	}
	switch cmd {
	case "query":
		return printHits(out, ix.Query(strings.Join(positional, " "), *k, *budget, *skeleton), *skeleton)
	case "path":
		if *directed {
			paths, err := ix.Trace(positional[0], positional[1], 1)
			if err != nil {
				return err
			}
			if len(paths) == 0 {
				_, _ = fmt.Fprintln(out, "no path")
				return nil
			}
			_, _ = fmt.Fprintln(out, strings.Join(paths[0], " -> "))
			return nil
		}
		path, err := ix.ShortestPath(positional[0], positional[1])
		if err != nil {
			return err
		}
		if path == nil {
			_, _ = fmt.Fprintln(out, "no path")
			return nil
		}
		_, _ = fmt.Fprintln(out, strings.Join(path, " -- "))
		return nil
	case "explain":
		sym, body, neigh, err := ix.Node(positional[0], 12)
		if err != nil {
			return err
		}
		_, _ = fmt.Fprintf(out, "%s  %s:%d\n\n%s\n\nneighbours:\n", sym.Name, sym.File, sym.Line, body)
		for _, n := range neigh {
			_, _ = fmt.Fprintf(out, "  %s  %s:%d\n", n.Name, n.File, n.Line)
		}
		return nil
	case "affected":
		syms, err := ix.Affected(positional[0], *depth, *limit)
		if err != nil {
			return err
		}
		for _, s := range syms {
			_, _ = fmt.Fprintf(out, "%s  %s:%d\n", s.Name, s.File, s.Line)
		}
		return nil
	}
	return fmt.Errorf("%w: unknown command %q", errUsage, cmd)
}

// splitArgs separates positional arguments from flags, so flags may come
// before or after the arguments, as in Graphify's CLI.
func splitArgs(args []string) (positional, flags []string) {
	for i := 0; i < len(args); i++ {
		a := args[i]
		if strings.HasPrefix(a, "-") && a != "-" {
			flags = append(flags, a)
			if !strings.Contains(a, "=") && i+1 < len(args) && !isBoolFlag(a) {
				i++
				flags = append(flags, args[i])
			}
			continue
		}
		positional = append(positional, a)
	}
	return positional, flags
}

func isBoolFlag(a string) bool {
	switch strings.TrimLeft(a, "-") {
	case "no-cache", "skeleton", "directed":
		return true
	}
	return false
}

func printHits(out io.Writer, hits []codegraph.Hit, skeleton bool) error {
	if len(hits) == 0 {
		_, _ = fmt.Fprintln(out, "no matches")
		return nil
	}
	for _, h := range hits {
		_, _ = fmt.Fprintf(out, "%s  %s:%d  score %.2f\n", h.Symbol.Name, h.Symbol.File, h.Symbol.Line, h.Score)
		if !skeleton && h.Body != "" {
			_, _ = fmt.Fprintln(out, h.Body)
			_, _ = fmt.Fprintln(out)
		}
	}
	return nil
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
