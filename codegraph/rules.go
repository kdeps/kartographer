package codegraph

import (
	_ "embed"

	"gopkg.in/yaml.v3"
)

//go:embed rules.yaml
var rulesYAML []byte

// rulesFile mirrors rules.yaml.
type rulesFile struct {
	Ranking struct {
		NameWeight   float64  `yaml:"name_weight"`
		PathWeight   float64  `yaml:"path_weight"`
		DegreeWeight float64  `yaml:"degree_weight"`
		BodyLines    int      `yaml:"body_lines"`
		Stopwords    []string `yaml:"stopwords"`
	} `yaml:"ranking"`
	Limits struct {
		FindSymbol     int `yaml:"find_symbol"`
		Callers        int `yaml:"callers"`
		Callees        int `yaml:"callees"`
		Impact         int `yaml:"impact"`
		ImpactSeeds    int `yaml:"impact_seeds"`
		TestsFor       int `yaml:"tests_for"`
		RankFiles      int `yaml:"rank_files"`
		FileNeighbors  int `yaml:"file_neighbors"`
		QueryBudget    int `yaml:"query_budget"`
		QueryK         int `yaml:"query_k"`
		MaxResult      int `yaml:"max_result"`
		NodeNeighbours int `yaml:"node_neighbours"`
		FindSeeds      int `yaml:"find_seeds"`
		RenderDepth    int `yaml:"render_depth"`
		RenderNodes    int `yaml:"render_nodes"`
		RenderEdges    int `yaml:"render_edges"`
		Recall         int `yaml:"recall"`
		MemoriesAbout  int `yaml:"memories_about"`
		ImpactFiles    int `yaml:"impact_files"`
		FetchBytes     int `yaml:"fetch_bytes"`
	} `yaml:"limits"`
	Paths struct {
		MaxDepth int `yaml:"max_depth"`
		MaxPaths int `yaml:"max_paths"`
	} `yaml:"paths"`
}

// rules are the code graph's business rules, read once from rules.yaml.
var rules, stopwords = loadRules() //nolint:gochecknoglobals // loaded from embedded YAML

func loadRules() (rulesFile, map[string]bool) {
	var r rulesFile
	if err := yaml.Unmarshal(rulesYAML, &r); err != nil {
		panic("codegraph: rules.yaml: " + err.Error())
	}
	set := make(map[string]bool, len(r.Ranking.Stopwords))
	for _, w := range r.Ranking.Stopwords {
		set[w] = true
	}
	return r, set
}
