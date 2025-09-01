package graph

import (
	"bytes"
	"sort"
	"testing"

	"github.com/charmbracelet/log"
	"github.com/spf13/afero"
)

func TestInvertDependencies(t *testing.T) {
	fs := afero.NewMemMapFs()
	var buf bytes.Buffer
	logger := log.NewWithOptions(&buf, log.Options{})
	dependencies := map[string][]string{
		"A": {"B", "C"},
		"B": {"D"},
		"C": {"D"},
		"D": {},
	}
	dg := NewDependencyGraph(fs, logger, dependencies)
	inverted := dg.InvertDependencies()
	expected := map[string][]string{
		"B": {"A"},
		"C": {"A"},
		"D": {"B", "C"},
	}
	for k, v := range expected {
		got := inverted[k]
		sort.Strings(got)
		sort.Strings(v)
		if !equal(got, v) {
			t.Errorf("Expected %v, got %v", v, got)
		}
	}
}

func TestInvertDependencies_Circular(t *testing.T) {
	fs := afero.NewMemMapFs()
	var buf bytes.Buffer
	logger := log.NewWithOptions(&buf, log.Options{})
	dependencies := map[string][]string{
		"A": {"B", "C"},
		"B": {"D"},
		"C": {"D"},
		"D": {"A"}, // Circular dependency here
	}
	dg := NewDependencyGraph(fs, logger, dependencies)
	inverted := dg.InvertDependencies()
	expected := map[string][]string{
		"B": {"A"},
		"C": {"A"},
		"D": {"B", "C"},
		"A": {"D"},
	}
	for k, v := range expected {
		got := inverted[k]
		sort.Strings(got)
		sort.Strings(v)
		if !equal(got, v) {
			t.Errorf("Expected %v, got %v", v, got)
		}
	}
}
