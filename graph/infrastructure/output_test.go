package infrastructure

import (
	"testing"
	"github.com/charmbracelet/log"
	"os"
)

func TestNewLoggerOutputWriter(t *testing.T) {
	logger := log.New(os.Stdout)
	writer := NewLoggerOutputWriter(logger)
	if writer == nil {
		t.Error("Expected writer to be created")
	}
}

func TestWriteLine(t *testing.T) {
	logger := log.New(os.Stdout)
	writer := NewLoggerOutputWriter(logger).(*LoggerOutputWriter)
	
	writer.WriteLine("test content")
}