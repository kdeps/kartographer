package community

import (
	"reflect"
	"testing"
)

func TestMethod(t *testing.T) {
	edges := []Edge{{0, 1, 1}, {1, 2, 1}, {3, 4, 1}}
	for name, want := range map[string]Detector{"louvain": Detect, "leiden": Leiden} {
		d, err := Method(name)
		if err != nil {
			t.Fatal(err)
		}
		gotA, gotQ := d(5, edges)
		wantA, wantQ := want(5, edges)
		if !reflect.DeepEqual(gotA, wantA) || gotQ != wantQ {
			t.Errorf("%s does not match its detector", name)
		}
	}
	if _, err := Method("girvan"); err == nil {
		t.Error("an unknown method is an error")
	}
}
