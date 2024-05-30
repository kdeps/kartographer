package graph

import (
	"github.com/charmbracelet/log"
	"github.com/spf13/afero"
)

type DependencyGraph struct {
	Fs               afero.Fs
	NodeDependencies map[string][]string
	DependencyGraph  []string
	VisitedPaths     map[string]bool
	Logger           *log.Logger
}

func NewDependencyGraph(fs afero.Fs, logger *log.Logger, dependencies map[string][]string) *DependencyGraph {
	return &DependencyGraph{
		Fs:               fs,
		NodeDependencies: dependencies,
		VisitedPaths:     make(map[string]bool),
		Logger:           logger,
	}
}
