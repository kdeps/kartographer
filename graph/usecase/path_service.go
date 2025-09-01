package usecase

import "github.com/kdeps/kartographer/graph/domain"

type PathServiceImpl struct {
	formatter domain.PathFormatter
	writer    domain.OutputWriter
}

func NewPathService(formatter domain.PathFormatter, writer domain.OutputWriter) domain.PathService {
	return &PathServiceImpl{
		formatter: formatter,
		writer:    writer,
	}
}

func (s *PathServiceImpl) ConstructPath(nodes []string, direction string) string {
	path := domain.NewPath(nodes, direction)
	return s.formatter.FormatPath(path)
}

func (s *PathServiceImpl) PrintPath(nodes []string, direction string) {
	pathStr := s.ConstructPath(nodes, direction)
	s.writer.WriteLine(pathStr)
}