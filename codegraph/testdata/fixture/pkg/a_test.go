package fx

import "testing"

func TestEntry(t *testing.T) {
	if Entry() != 3 {
		t.Fatal("bad")
	}
}
