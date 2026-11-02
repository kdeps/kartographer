package codegraph

import (
	"os"
	"testing"

	"gopkg.in/yaml.v3"
)

// TestMain configures the rules the tests run with. The values are the ones
// kdeps ships in its rules file.
func TestMain(m *testing.M) {
	raw, err := yaml.Marshal(testRules())
	if err != nil {
		panic(err)
	}
	if err := Configure(raw); err != nil {
		panic(err)
	}
	os.Exit(m.Run())
}

func testRules() rulesFile {
	var r rulesFile
	r.Paths.SkipDirs = []string{"vendor", "node_modules", "testdata"}
	r.Paths.SkipHidden = true
	r.Ranking.NameWeight = 3
	r.Ranking.PathWeight = 1
	r.Ranking.DegreeWeight = 0.1
	r.Ranking.BodyLines = 30
	r.Ranking.Stopwords = []string{"the", "a", "an", "of", "is", "are", "where", "how", "what", "which",
		"does", "do", "in", "to", "and", "or", "for", "on", "with", "by", "it", "this", "that", "be", "when", "who"}
	r.Limits.FindSymbol = 30
	r.Limits.Callers = 25
	r.Limits.Callees = 25
	r.Limits.Impact = 60
	r.Limits.ImpactSeeds = 8
	r.Limits.TestsFor = 25
	r.Limits.RankFiles = 15
	r.Limits.FileNeighbors = 50
	r.Limits.QueryBudget = 5000
	r.Limits.QueryK = 5
	r.Limits.MaxResult = 300
	r.Limits.NodeNeighbours = 12
	r.Limits.FindSeeds = 10
	r.Limits.RenderDepth = 1
	r.Limits.RenderNodes = 300
	r.Limits.RenderEdges = 4000
	r.Limits.Recall = 10
	r.Limits.MemoriesAbout = 5
	r.Limits.ImpactFiles = 15
	r.Limits.FetchBytes = 16384
	r.Limits.References = 25
	r.Limits.ImportsExports = 50
	r.Paths.MaxDepth = 8
	r.Paths.MaxPaths = 10
	return r
}
