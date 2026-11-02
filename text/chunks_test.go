package text

import (
	"reflect"
	"strings"
	"testing"
)

func TestChunks(t *testing.T) {
	if Chunks("  ", 10) != nil {
		t.Error("blank text has no pieces")
	}
	if got := Chunks("short", 10); !reflect.DeepEqual(got, []string{"short"}) {
		t.Errorf("short = %v", got)
	}
	para := strings.Repeat("a", 7) + "\n\n" + strings.Repeat("b", 7)
	if got := Chunks(para, 12); !reflect.DeepEqual(got, []string{"aaaaaaa", "bbbbbbb"}) {
		t.Errorf("cut at the paragraph break: %v", got)
	}
	if got := Chunks(strings.Repeat("c", 25), 10); len(got) != 3 || got[0] != strings.Repeat("c", 10) {
		t.Errorf("hard cut without a break: %v", got)
	}
}
