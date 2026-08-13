package graph

import (
	"fmt"
	"sort"

	"github.com/charmbracelet/log"
	"github.com/spf13/afero"

	"github.com/kdeps/kartographer/graph/index/domain"
	"github.com/kdeps/kartographer/graph/index/infrastructure"
	"github.com/kdeps/kartographer/graph/index/usecase"
)

// IndexedGraph indexes a folder into a persistent bbolt database and answers
// "show me the graph" queries for a given file or topic, reusing
// DependencyGraph's existing traversal/printing machinery.
type IndexedGraph struct {
	Fs     afero.Fs
	Logger *log.Logger

	store *infrastructure.BoltIndexStore
	idx   *usecase.IndexerService
}

// DefaultIndexExtensions is the file extension allowlist used when none is
// supplied to IndexFolder.
var DefaultIndexExtensions = []string{".md", ".markdown", ".txt", ".yaml", ".yml"} //nolint:gochecknoglobals

// NewIndexedGraph opens (or creates) a bbolt database at dbPath for indexing
// and querying. The caller must call Close() when done.
func NewIndexedGraph(fs afero.Fs, logger *log.Logger, dbPath string) (*IndexedGraph, error) {
	store, err := infrastructure.NewBoltIndexStore(dbPath)
	if err != nil {
		return nil, err
	}

	return &IndexedGraph{
		Fs:     fs,
		Logger: logger,
		store:  store,
	}, nil
}

func (ig *IndexedGraph) Close() error {
	return ig.store.Close()
}

// IndexFolder walks root and (re)indexes every matching file into the bbolt
// database. Extensions defaults to DefaultIndexExtensions when nil.
func (ig *IndexedGraph) IndexFolder(root string, extensions []string) error {
	if extensions == nil {
		extensions = DefaultIndexExtensions
	}

	walker := infrastructure.NewAferoFileWalker(ig.Fs)
	refExtr := infrastructure.NewLinkReferenceExtractor(root)
	topicExtr := infrastructure.NewFrontmatterTopicExtractor()

	ig.idx = usecase.NewIndexerService(ig.Fs, walker, refExtr, topicExtr, ig.store)
	return ig.idx.IndexFolder(root, extensions)
}

func (ig *IndexedGraph) indexer() (*usecase.IndexerService, domain.IndexStore) {
	if ig.idx == nil {
		ig.idx = usecase.NewIndexerService(ig.Fs, nil, nil, nil, ig.store)
	}
	return ig.idx, ig.store
}

// ShowFile prints the reference tree rooted at path, followed by every other
// indexed file that shares at least one topic with it.
func (ig *IndexedGraph) ShowFile(path string) error {
	idx, _ := ig.indexer()

	refGraph, err := idx.BuildReferenceGraph()
	if err != nil {
		return err
	}

	dg := NewDependencyGraph(ig.Fs, ig.Logger, refGraph)
	fmt.Printf("References for %s:\n", path)
	dg.ListDependencyTree(path)

	related, err := idx.FilesRelatedByTopic(path)
	if err != nil {
		return err
	}
	fmt.Println("Related by topic:")
	if len(related) == 0 {
		fmt.Println("  (none)")
		return nil
	}
	sort.Strings(related)
	for _, r := range related {
		fmt.Printf("  %s\n", r)
	}
	return nil
}

// ShowTopic prints every indexed file tagged with topic, then each of those
// files' own direct references, as a single tree rooted at the topic.
func (ig *IndexedGraph) ShowTopic(topic string) error {
	idx, _ := ig.indexer()

	files, err := idx.FilesForTopic(topic)
	if err != nil {
		return err
	}

	refGraph, err := idx.BuildReferenceGraph()
	if err != nil {
		return err
	}

	rootNode := "topic:" + topic
	merged := make(map[string][]string, len(refGraph)+1)
	for k, v := range refGraph {
		merged[k] = v
	}
	merged[rootNode] = files

	dg := NewDependencyGraph(ig.Fs, ig.Logger, merged)
	fmt.Printf("Files tagged %q:\n", topic)
	dg.ListDependencyTree(rootNode)
	return nil
}

// ShowAll prints the reference tree for every root file in the index (files
// that nothing else references) -- "graph all the indexed data".
func (ig *IndexedGraph) ShowAll() error {
	idx, _ := ig.indexer()

	refGraph, err := idx.BuildReferenceGraph()
	if err != nil {
		return err
	}

	dg := NewDependencyGraph(ig.Fs, ig.Logger, refGraph)
	referenced := dg.InvertDependencies()

	var roots []string
	for node := range refGraph {
		if _, isReferenced := referenced[node]; !isReferenced {
			roots = append(roots, node)
		}
	}
	sort.Strings(roots)

	for _, root := range roots {
		fmt.Printf("References for %s:\n", root)
		dg.ListDependencyTree(root)
	}
	return nil
}
