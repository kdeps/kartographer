package graph

import (
	"os"
	"testing"
)

func TestEqual(t *testing.T) {
	tests := []struct {
		name     string
		a        []string
		b        []string
		expected bool
	}{
		{
			name:     "equal arrays",
			a:        []string{"A", "B", "C"},
			b:        []string{"A", "B", "C"},
			expected: true,
		},
		{
			name:     "different lengths",
			a:        []string{"A", "B"},
			b:        []string{"A", "B", "C"},
			expected: false,
		},
		{
			name:     "different values",
			a:        []string{"A", "B", "C"},
			b:        []string{"A", "X", "C"},
			expected: false,
		},
		{
			name:     "empty arrays",
			a:        []string{},
			b:        []string{},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := equal(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("equal(%v, %v) = %v, want %v", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}

func TestCreatePipe(t *testing.T) {
	// Test normal operation
	r, w := createPipe()
	if r == nil || w == nil {
		t.Error("Expected valid pipe files")
	}
	r.Close()
	w.Close()
}

// MockPipeCreator for testing error conditions
type MockPipeCreator struct {
	shouldError bool
}

func (m *MockPipeCreator) CreatePipe() (*os.File, *os.File, error) {
	if m.shouldError {
		return nil, nil, os.ErrInvalid
	}
	return os.Pipe()
}

func TestCreatePipeWithCreator_Error(t *testing.T) {
	mock := &MockPipeCreator{shouldError: true}
	
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic on pipe creation error")
		}
	}()
	
	createPipeWithCreator(mock)
}

func TestCreatePipeWithCreator_Success(t *testing.T) {
	mock := &MockPipeCreator{shouldError: false}
	r, w := createPipeWithCreator(mock)
	
	if r == nil || w == nil {
		t.Error("Expected valid pipe files")
	}
	r.Close()
	w.Close()
}