package notes

import (
	"reflect"
	"testing"
)

func TestListItems(t *testing.T) {
	for in, want := range map[string]string{
		"- a": "a", "  * b ": "b", "2. c": "c", "3) d": "d", "- [x] done": "done", "[ ] open": "open", "plain": "plain",
	} {
		if got := ListItem(in); got != want {
			t.Fatalf("ListItem(%q) = %q, want %q", in, got, want)
		}
	}
	got := ListItems("- one\n\n  - [ ] two\n- \n3. three")
	if want := []string{"one", "two", "three"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("ListItems = %q, want %q", got, want)
	}
}
