package graph

import "testing"

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