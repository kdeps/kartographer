package infrastructure

import (
	"fmt"
	"github.com/charmbracelet/log"
	"github.com/kdeps/kartographer/graph/domain"
)

type LoggerOutputWriter struct {
	logger *log.Logger
}

func NewLoggerOutputWriter(logger *log.Logger) domain.OutputWriter {
	return &LoggerOutputWriter{
		logger: logger,
	}
}

func (w *LoggerOutputWriter) WriteLine(content string) {
	fmt.Println(content)
}