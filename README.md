# kartographer
**A Graph Library for Resolving Dependent Nodes**

This package provides functionality for traversing and analyzing a dependency graph. The `DependencyGraph` struct and
its methods allow for listing dependencies in various orders, including direct, recursive, top-down, and reverse
dependencies.

## Table of Contents

- [Installation](#installation)
- [Usage](#usage)
- [Functions](#functions)
  - [TraverseDependencyGraph](#traversedependencygraph)
  - [ListDependencyTreeTopDown](#listdependencytreetopdown)
  - [ListDependenciesRecursive](#listdependenciesrecursive)
  - [ListReverseDependencies](#listreversedependencies)
  - [ListDependencyTree](#listdependencytree)
  - [BuildDependencyStack](#builddependencystack)
  - [InvertDependencies](#invertdependencies)
  - [ListDirectDependencies](#listdirectdependencies)
- [Utility Functions](#utility-functions)
- [Indexing a Folder](#indexing-a-folder)

## Installation

To use this package, you need to import it into your Go project. Make sure you have the following dependencies in your
`go.mod` file:

```go
import (
  graph "github.com/kdeps/kartographer/graph"
)
```

## Usage

First, create a new `DependencyGraph` instance by calling the `NewDependencyGraph` function with the appropriate parameters.

```go
import (
    graph "github.com/kdeps/kartographer/graph"
)

dependencies := map[string][]string{
    "A": {"B", "C"},
    "B": {"D"},
    "C": {"D"},
    "D": {},
}

dg := graph.NewDependencyGraph(fs, logger, dependencies)
```

Then you can call any of the methods on the `DependencyGraph` instance to list dependencies in various orders.

```go
dg.ListDirectDependencies("A")
dg.ListDependencyTree("A")
dg.ListDependencyTreeTopDown("A")
dg.ListReverseDependencies("D")

```

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

Two kinds of edges are derived automatically while indexing:

- **References** — explicit markdown links (`[text](path)`) and wikilinks (`[[path]]`) found in a file's content,
  resolved relative to the file and kept only if they resolve to another file inside the indexed root.
- **Topics** — a `topics:` or `tags:` list declared in a leading YAML frontmatter block (`---\n...\n---`). Files
  sharing a topic are considered related.

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
