package graph

import (
	"io"
	"testing"

	"github.com/charmbracelet/log"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

// diamond is A -> B, A -> C, B -> D, C -> D.
func diamond() *DependencyGraph {
	return NewDependencyGraph(afero.NewMemMapFs(), log.New(io.Discard), map[string][]string{
		"A": {"B", "C"},
		"B": {"D"},
		"C": {"D"},
		"D": {},
	})
}

func TestDirectReturnsOnlyImmediateDependencies(t *testing.T) {
	assert.Equal(t, []string{"B", "C"}, diamond().Direct("A"))
	assert.Empty(t, diamond().Direct("missing"))
}

func TestRecursiveWalksEveryDependencyOnce(t *testing.T) {
	assert.Equal(t, []string{"B", "D", "C", "D"}, diamond().Recursive("A"),
		"D is listed on each path that reaches it, but its own dependencies are walked once")
	assert.Equal(t, []string{}, diamond().Recursive("D"))
}

func TestReverseReturnsTheNodesThatDependOnIt(t *testing.T) {
	assert.ElementsMatch(t, []string{"B", "C"}, diamond().Reverse("D"))
	assert.Empty(t, diamond().Reverse("A"))
}

func TestStackPutsDependenciesFirst(t *testing.T) {
	assert.Equal(t, []string{"D", "B", "C", "A"}, diamond().Stack("A"))
}

func TestPathsReportsEachDistinctPathOnce(t *testing.T) {
	assert.Equal(t, [][]string{
		{"A"},
		{"A", "B"},
		{"A", "B", "D"},
		{"A", "C"},
	}, diamond().Paths("A"))
}
