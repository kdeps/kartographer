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
		wantA, wantQ := want(5, edges)
		for range 200 {
			gotA, gotQ := d(5, edges)
			if !reflect.DeepEqual(gotA, wantA) || gotQ != wantQ {
				t.Fatalf("%s does not match its detector on every call: %v %v vs %v %v", name, gotA, gotQ, wantA, wantQ)
			}
		}
	}
	if _, err := Method("girvan"); err == nil {
		t.Error("an unknown method is an error")
	}
}
