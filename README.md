# kartographer

Kartographer builds a code graph of a module: symbols, call edges, references, and test links, queried from the command line or over MCP. Go is analysed with `go/types`, SSA and call-graph analysis. Other languages use the language server installed for them.

## Install

```bash
go install github.com/kdeps/kartographer/cmd/kartographer@main
go install github.com/kdeps/kartographer/cmd/kartographer-mcp@main
```

## Rules file

Every command needs a rules file (`-rules`). Kartographer has no built-in thresholds: ranking weights, result limits and stopwords come from YAML that the caller supplies. kdeps ships one at `assets/codegraph/rules.yaml`.

```yaml
ranking:
  name_weight: 3
  path_weight: 1
  degree_weight: 0.1
  body_lines: 30
  stopwords: [the, of, is]
paths:
  max_depth: 8
  max_paths: 10
limits:
  max_result: 300
```

## Command line

```bash
kartographer -rules rules.yaml update [dir]                  # build or refresh the index
kartographer -rules rules.yaml query "where is the retry"    # matching symbols with source
kartographer -rules rules.yaml path "Entry" "leaf"           # shortest connection
kartographer -rules rules.yaml path "Entry" "leaf" --directed
kartographer -rules rules.yaml explain "helper"              # source and neighbours
kartographer -rules rules.yaml affected "leaf" --depth 2     # dependents, nearest first
kartographer -rules rules.yaml watch [dir] [-interval 2s]    # rebuild when files change
kartographer -rules rules.yaml hook install                  # rebuild after commits and checkouts
kartographer -rules rules.yaml hook uninstall|status
kartographer -rules rules.yaml export graphml [-o FILE]      # graphml, html, svg, neo4j, falkordb
kartographer -rules rules.yaml export obsidian -o vault/     # one note per symbol, linked by calls
kartographer -rules rules.yaml export wiki -o site/          # index plus one page per file
kartographer -rules rules.yaml add "https://example.com/a"   # fetch a page into ./raw
```

Global flags: `-rules FILE` (required), `-dir DIR` (module root, default `.`), `-no-cache`.
`callers` and `callees` rows, and the GraphML and Cypher exports, carry a `confidence` of `EXTRACTED` (a direct call site or language server reference) or `INFERRED` (an interface method call that VTA resolved by type propagation, so the target is an approximation).

`query` takes `--k`, `--budget` and `--skeleton`. `add` takes `--dir-raw` (default `raw`), `--author` and `--contributor`; it uses `--dir-raw` rather than `--dir` because `-dir` selects the module root.

Exit codes: 0 success, 1 a lookup or rules error, 2 bad usage.

The git hooks rebuild only when `KARTOGRAPHER_RULES` is set to the rules file path. They never guess one.

## MCP server

```bash
kartographer-mcp -rules rules.yaml -dir /path/to/module
```

Serves the code graph over stdio MCP. The tools are:

| Group | Tools |
|---|---|
| Symbols | `find_symbol`, `node`, `search`, `find_seeds`, `query`, `rank_files` |
| Calls | `callers`, `callees`, `trace`, `shortest_path`, `expand`, `render_subgraph` |
| Impact and tests | `impact`, `impact_and_risk`, `tests_for` |
| Files and references | `file_neighbors`, `references` (every language with a server), `imports_exports` (Go only) |
| Memory | `remember`, `recall`, `memories_about` |
| Workspace | `list_modules`, `list_workspaces`, `set_workspace`, `stats`, `fetch` |

## Index cache

`BuildCached` saves the index under `os.UserCacheDir()/kartographer/index`. It is reused until a file's path, size or modification time changes. `-no-cache` rebuilds it.

## Language support

| Language | How |
|---|---|
| Go | `go/types`, SSA and call-graph analysis. Supports references and imports. |
| Other | The language server listed in `lsp/registry.yaml` for the file's extension, if it is on PATH. Its call hierarchy supplies the call edges, and its references answers supply the non-call uses. Both are read at index time, one request per function, so indexing a large module with a server takes longer. |

`lsp/registry.yaml` lists 13 languages, from C and C++ to Swift and C#. The first server on PATH for a language is used.

## Dependency graph package

The `graph` package is a separate, lower-level library: a dependency graph over string keys, with direct, recursive, top-down, reverse and stack-based traversals. It does not read source code.

## Functions

### TraverseDependencyGraph

Traverses the dependency graph starting from a given node and prints the paths.

```go
func (dg *DependencyGraph) TraverseDependencyGraph(node string, dependencies map[string][]string, visited map[string]bool)
```

### ListDependencyTreeTopDown

Lists the dependency tree in a top-down order starting from a given node.

```go
func (dg *DependencyGraph) ListDependencyTreeTopDown(node string)
```

### ListDependenciesRecursive

Lists all dependencies recursively from a given node and prints each dependency path.

```go
func (dg *DependencyGraph) ListDependenciesRecursive(node string, path []string, visited map[string]bool)
```

### ListReverseDependencies

Lists all reverse dependencies for a given node.

```go
func (dg *DependencyGraph) ListReverseDependencies(node string)
```

### ListDependencyTree

Lists the entire dependency tree for a given node.

```go
func (dg *DependencyGraph) ListDependencyTree(node string)
```

### BuildDependencyStack

Builds and returns a stack of dependencies starting from a given node.

```go
func (dg *DependencyGraph) BuildDependencyStack(node string, visited map[string]bool) []string
```

### InvertDependencies

Inverts the dependency graph, swapping dependencies with their dependents.

```go
func (dg *DependencyGraph) InvertDependencies() map[string][]string
```

### ListDirectDependencies

Lists direct dependencies for a given node.

```go
func (dg *DependencyGraph) ListDirectDependencies(node string)
```

## Utility Functions

The package also includes utility functions for constructing or printing dependency paths.

### ConstructDependencyPath

Construct a given dependency path which returns a string

```go
func (dg *DependencyGraph) ConstructDependencyPath(path []string, dir string) string
```

### PrintDependencyPath

Prints a given dependency path.

```go
func (dg *DependencyGraph) PrintDependencyPath(path []string, dir string)
```

This function can be used to format and print paths in a human-readable form.

---

With this package, you can easily manage and traverse dependency graphs in Go, allowing for complex dependency analysis
and visualization.

## Indexing a Folder

`IndexedGraph` indexes an existing folder into a persistent [bbolt](https://github.com/etcd-io/bbolt) database and
lets you query the resulting graph by file or by topic, reusing the same tree-printing machinery as
`DependencyGraph`.

Two kinds of edges are derived automatically while indexing, using a per-file-extension extractor:

- **References** — resolved relative to the file, kept only if they resolve to another file inside the indexed root
  (a link outside the root, or to an external URL/stdlib/third-party package, is dropped, not guessed at):
  - `.md` / `.markdown` / `.txt` — markdown links (`[text](path)`) and wikilinks (`[[path]]`)
  - `.html` / `.htm` — `href`/`src` attribute values
  - Source code — import/include statements, both relative (`from . import x`, `#include "local.h"`) and
    non-relative/module-style (`import "github.com/x/pkg"`, `import foo.bar`), the latter resolved by matching
    against the files actually present in the indexed root. Covers Go, Python, Rust, TypeScript/JavaScript, C/C++,
    Ruby, and Java. This is heuristic regex extraction, not a full parser — it won't catch every construct a
    language's import system supports, and ambiguous non-relative matches (multiple files with the same name) are
    dropped rather than guessed.
  - `.json` / `.yaml` / `.yml` (bare, not paired with a `---` frontmatter block) — no reference extraction; too
    ambiguous which string values are paths.
- **Topics** — a `topics:` or `tags:` list, either declared in a leading YAML frontmatter block (`---\n...\n---`,
  any file type) or, for a bare `.yaml`/`.yml`/`.json` file with no frontmatter markers, a top-level key in the
  whole document. Files sharing a topic are considered related.

The default extension allowlist (`.md`, `.markdown`, `.txt`, `.yaml`, `.yml`) is unchanged — source code, HTML, and
JSON support is opt-in via an explicit `extensions` argument to `IndexFolder`, so indexing a docs folder never
silently starts walking an entire adjacent source tree.

```go
import (
    "github.com/charmbracelet/log"
    "github.com/spf13/afero"
    graph "github.com/kdeps/kartographer/graph"
)

ig, err := graph.NewIndexedGraph(afero.NewOsFs(), log.Default(), "index.db")
if err != nil {
    // handle error
}
defer ig.Close()

// Walk ./docs and index every .md/.markdown/.txt/.yaml/.yml file.
// Pass a non-nil []string to override the default extension allowlist.
// Returns the number of files indexed.
n, err := ig.IndexFolder("./docs", nil)
if err != nil {
    // handle error
}

// Show the reference tree for a single file, plus every file that shares a topic with it.
ig.ShowFile("docs/intro.md")

// Show every file tagged with a given topic, and each of those files' own references.
ig.ShowTopic("getting-started")

// Show the reference tree for every root file in the index (files nothing else references) --
// i.e. graph everything that's been indexed.
ig.ShowAll()
```

`Show*` print to stdout via the logger. For structured (non-printing) access to the same data — e.g. to feed a UI or an API response — use the `Graph*` counterparts, which return the data directly instead of printing it:

```go
// references is the full indexed reference graph (map[file][]references).
// relatedByTopic is every other file that shares a topic with docs/intro.md.
references, relatedByTopic, err := ig.GraphFile("docs/intro.md")

// files is every file tagged "getting-started"; references is the full reference graph.
files, references, err := ig.GraphTopic("getting-started")

// references is the full reference graph; roots is every file nothing else references.
references, roots, err := ig.GraphAll()
```

Re-running `IndexFolder` on the same database fully re-indexes the folder, overwriting prior records for files that
still exist and updating the topic index accordingly.
