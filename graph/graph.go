package graph

import (
	"github.com/charmbracelet/log"
	"github.com/spf13/afero"
	"github.com/kdeps/kartographer/graph/domain"
	"github.com/kdeps/kartographer/graph/infrastructure"
	"github.com/kdeps/kartographer/graph/usecase"
)

type DependencyGraph struct {
	Fs               afero.Fs
	NodeDependencies map[string][]string
	DependencyGraph  []string
	VisitedPaths     map[string]bool
	Logger           *log.Logger
	depService       domain.DependencyService
	pathService      domain.PathService
}

func NewDependencyGraph(fs afero.Fs, logger *log.Logger, dependencies map[string][]string) *DependencyGraph {
	repo := infrastructure.NewInMemoryGraphRepository(dependencies)
	formatter := infrastructure.NewArrowPathFormatter()
	writer := infrastructure.NewLoggerOutputWriter(logger)
	pathService := usecase.NewPathService(formatter, writer)
	depService := usecase.NewDependencyService(repo, pathService)
	return &DependencyGraph{
		Fs:               fs,
		NodeDependencies: dependencies,
		VisitedPaths:     make(map[string]bool),
		Logger:           logger,
		depService:       depService,
		pathService:      pathService,
	}
}
