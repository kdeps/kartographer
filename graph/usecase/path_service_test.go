package usecase

import (
	"testing"
	"github.com/kdeps/kartographer/graph/infrastructure"
)

func TestNewPathService(t *testing.T) {
	formatter := infrastructure.NewArrowPathFormatter()
	writer := &mockOutputWriter{}
	
	service := NewPathService(formatter, writer)
	if service == nil {
		t.Error("Expected service to be created")
	}
}

func TestConstructPath(t *testing.T) {
	formatter := infrastructure.NewArrowPathFormatter()
	writer := &mockOutputWriter{}
	service := NewPathService(formatter, writer).(*PathServiceImpl)
	
	result := service.ConstructPath([]string{"A", "B", "C"}, "forward")
	expected := "A -> B -> C"
	
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestPrintPath(t *testing.T) {
	formatter := infrastructure.NewArrowPathFormatter()
	writer := &mockOutputWriter{}
	service := NewPathService(formatter, writer).(*PathServiceImpl)
	
	service.PrintPath([]string{"A", "B"}, "forward")
	
	if len(writer.lines) != 1 {
		t.Errorf("Expected 1 line printed, got %d", len(writer.lines))
	}
	if writer.lines[0] != "A -> B" {
		t.Errorf("Expected 'A -> B', got '%s'", writer.lines[0])
	}
}