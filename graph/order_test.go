package graph

import (
	"errors"
	"io"
	"reflect"
	"testing"

	"github.com/charmbracelet/log"
	"github.com/spf13/afero"
)

func newTestGraph(deps map[string][]string) *DependencyGraph {
	return NewDependencyGraph(afero.NewMemMapFs(), log.New(io.Discard), deps)
}

func TestOrder(t *testing.T) {
	g := newTestGraph(map[string][]string{
		"app": {"db", "cache"}, "db": {"config"}, "cache": {"config"}, "config": nil, "other": nil,
	})
	all, err := g.Order()
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"config", "db", "cache", "app", "other"}
	if !reflect.DeepEqual(all, want) {
		t.Errorf("all = %v, want %v", all, want)
	}
	sub, err := g.Order("db")
	if err != nil || !reflect.DeepEqual(sub, []string{"config", "db"}) {
		t.Errorf("a root orders only what it needs: %v, %v", sub, err)
	}
	again, _ := g.Order()
	if !reflect.DeepEqual(all, again) {
		t.Errorf("the order is the same every call: %v, %v", all, again)
	}
}

func TestOrder_Cycle(t *testing.T) {
	g := newTestGraph(map[string][]string{"a": {"b"}, "b": {"c"}, "c": {"a"}, "d": nil})
	_, err := g.Order()
	var ce *CycleError
	if !errors.As(err, &ce) || ce.Node == "" {
		t.Fatalf("want a *CycleError, got %v", err)
	}
	if _, err := g.Order("d"); err != nil {
		t.Errorf("a root off the cycle orders fine: %v", err)
	}
	self := newTestGraph(map[string][]string{"x": {"x"}})
	if _, err := self.Order(); !errors.As(err, &ce) || ce.Node != "x" {
		t.Errorf("a node needing itself is a cycle: %v", err)
	}
}
