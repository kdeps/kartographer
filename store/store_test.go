package store

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

type sample struct {
	Name string `json:"name"`
	N    int    `json:"n"`
}

func TestWriteReadRoundTrip(t *testing.T) {
	file := filepath.Join(t.TempDir(), "nested", "state.json")
	if err := WriteJSON(file, sample{Name: "graph", N: 3}); err != nil {
		t.Fatal(err)
	}
	var got sample
	if err := ReadJSON(file, &got); err != nil || got != (sample{Name: "graph", N: 3}) {
		t.Fatalf("round trip = %+v, %v", got, err)
	}
	if _, err := os.Stat(file + ".tmp"); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("temporary file left behind: %v", err)
	}
}

func TestReadMissingFile(t *testing.T) {
	var got sample
	if err := ReadJSON(filepath.Join(t.TempDir(), "absent.json"), &got); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("want not-exist, got %v", err)
	}
}

func TestReadBadJSON(t *testing.T) {
	file := filepath.Join(t.TempDir(), "bad.json")
	if err := os.WriteFile(file, []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	var got sample
	if err := ReadJSON(file, &got); err == nil {
		t.Error("bad JSON must fail")
	}
}

func TestWriteJSONFailsOnUnencodable(t *testing.T) {
	if err := WriteJSON(filepath.Join(t.TempDir(), "x.json"), make(chan int)); err == nil {
		t.Error("an unencodable value must fail")
	}
}
