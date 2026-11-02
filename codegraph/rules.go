package codegraph

import (
	"errors"

	"gopkg.in/yaml.v3"
)

// rulesFile is the shape of the business rules a caller supplies. The values
// are the caller's policy: the library holds none of its own.
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
		References     int `yaml:"references"`
		ImportsExports int `yaml:"imports_exports"`
	} `yaml:"limits"`
	Paths struct {
		MaxDepth int `yaml:"max_depth"`
		MaxPaths int `yaml:"max_paths"`
		// SkipDirs are folder names the Go file listing never enters.
		SkipDirs []string `yaml:"skip_dirs"`
		// SkipHidden skips folders whose names start with a dot.
		SkipHidden bool `yaml:"skip_hidden"`
	} `yaml:"paths"`
}

// ErrNotConfigured is returned by Build until Configure has been called.
var ErrNotConfigured = errors.New("codegraph: rules not configured; call Configure first")

// rules and stopwords are the configured policy. They are set once, by Configure.
var (
	rules      rulesFile       //nolint:gochecknoglobals // set once by Configure
	stopwords  map[string]bool //nolint:gochecknoglobals // set once by Configure
	configured bool            //nolint:gochecknoglobals // set once by Configure
)

// Configure sets the business rules from YAML the caller supplies (ranking
// weights, stopwords, result limits, path bounds). Build fails with
// ErrNotConfigured until it succeeds.
func Configure(data []byte) error {
	var r rulesFile
	if err := yaml.Unmarshal(data, &r); err != nil {
		return err
	}
	set := make(map[string]bool, len(r.Ranking.Stopwords))
	for _, w := range r.Ranking.Stopwords {
		set[w] = true
	}
	rules, stopwords, configured = r, set, true
	return nil
}
