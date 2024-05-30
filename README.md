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

## Installation

To use this package, you need to import it into your Go project. Make sure you have the following dependencies in your
`go.mod` file:

```go
require "github.com/kdeps/kartographer"
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
