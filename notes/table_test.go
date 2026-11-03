package notes

import (
	"reflect"
	"testing"
)

func TestSplitBody(t *testing.T) {
	body := "# Title\n\nSome prose.\nMore prose.\n\n| Source | Path |\n|---|:--|\n| repo | `/src/app` |\n| folder |\n"
	p := SplitBody(body, []string{"Source", "Path"})
	if p.ProseLines != 2 {
		t.Errorf("prose lines = %d", p.ProseLines)
	}
	want := [][]string{{"repo", "`/src/app`"}, {"folder"}}
	if !reflect.DeepEqual(p.Rows, want) {
		t.Errorf("rows = %q", p.Rows)
	}
	if got := SplitBody("| source | x |", []string{"Source"}); len(got.Rows) != 0 {
		t.Errorf("the header row matches without case: %q", got.Rows)
	}
	if got := SplitBody("| a | b |", nil); len(got.Rows) != 1 {
		t.Errorf("no header keeps every row: %q", got.Rows)
	}
}

func TestAppendTableRow(t *testing.T) {
	header := []string{"Source", "Path"}
	first := AppendTableRow("---\na: 1\n---\n\n", header, []string{"repo", "/r"}, true)
	if first != "---\na: 1\n---\n| Source | Path |\n|---|---|\n| repo | /r |\n" {
		t.Errorf("first row = %q", first)
	}
	second := AppendTableRow(first, header, []string{"folder", "/f"}, false)
	if second != first+"| folder | /f |\n" {
		t.Errorf("second row = %q", second)
	}
}

func TestTableRow(t *testing.T) {
	cells, ok := TableRow("  | search | /a/b |  ")
	if !ok || !reflect.DeepEqual(cells, []string{"search", "/a/b"}) {
		t.Fatalf("row = %q, %v", cells, ok)
	}
	if cells, ok = TableRow("||"); !ok || !reflect.DeepEqual(cells, []string{""}) {
		t.Fatalf("empty row = %q, %v", cells, ok)
	}
	for _, line := range []string{"not a row", "| open", "|", ""} {
		if _, ok = TableRow(line); ok {
			t.Fatalf("%q read as a row", line)
		}
	}
}
