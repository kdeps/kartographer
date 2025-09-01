package infrastructure

import (
	"testing"
	"github.com/kdeps/kartographer/graph/domain"
)

func TestNewArrowPathFormatter(t *testing.T) {
	formatter := NewArrowPathFormatter()
	if formatter == nil {
		t.Error("Expected formatter to be created")
	}
}

func TestFormatPath_Forward(t *testing.T) {
	formatter := NewArrowPathFormatter().(*ArrowPathFormatter)
	path := domain.NewPath([]string{"A", "B", "C"}, "forward")
	
	result := formatter.FormatPath(path)
	expected := "A -> B -> C"
	
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestFormatPath_Reverse(t *testing.T) {
	formatter := NewArrowPathFormatter().(*ArrowPathFormatter)
	path := domain.NewPath([]string{"A", "B", "C"}, "reverse")
	
	result := formatter.FormatPath(path)
	expected := "A <- B <- C"
	
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestFormatPath_EmptyNodes(t *testing.T) {
	formatter := NewArrowPathFormatter().(*ArrowPathFormatter)
	path := domain.NewPath([]string{}, "forward")
	
	result := formatter.FormatPath(path)
	expected := ""
	
	if result != expected {
		t.Errorf("Expected empty string, got '%s'", result)
	}
}