package infrastructure

import (
	"strings"
	"github.com/kdeps/kartographer/graph/domain"
)

type ArrowPathFormatter struct{}

func NewArrowPathFormatter() domain.PathFormatter {
	return &ArrowPathFormatter{}
}

func (f *ArrowPathFormatter) FormatPath(path *domain.Path) string {
	if len(path.Nodes) == 0 {
		return ""
	}
	if path.Direction == "reverse" {
		return f.formatReversePath(path.Nodes)
	}
	return f.formatForwardPath(path.Nodes)
}

func (f *ArrowPathFormatter) formatForwardPath(nodes []string) string {
	return strings.Join(nodes, " -> ")
}

func (f *ArrowPathFormatter) formatReversePath(nodes []string) string {
	return strings.Join(nodes, " <- ")
}