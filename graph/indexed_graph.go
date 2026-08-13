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
// database, returning the number of files indexed. Extensions defaults to
// DefaultIndexExtensions when nil.
func (ig *IndexedGraph) IndexFolder(root string, extensions []string) (int, error) {
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

// GraphFile returns the full indexed reference graph, plus every other
// indexed file that shares at least one topic with path.
func (ig *IndexedGraph) GraphFile(path string) (references map[string][]string, relatedByTopic []string, err error) {
	idx, _ := ig.indexer()

	references, err = idx.BuildReferenceGraph()
	if err != nil {
		return nil, nil, err
	}

	relatedByTopic, err = idx.FilesRelatedByTopic(path)
	if err != nil {
		return nil, nil, err
	}
	sort.Strings(relatedByTopic)
	return references, relatedByTopic, nil
}

// GraphTopic returns every indexed file tagged with topic, plus the full
// indexed reference graph.
func (ig *IndexedGraph) GraphTopic(topic string) (files []string, references map[string][]string, err error) {
	idx, _ := ig.indexer()

	files, err = idx.FilesForTopic(topic)
	if err != nil {
		return nil, nil, err
	}

	references, err = idx.BuildReferenceGraph()
	if err != nil {
		return nil, nil, err
	}
	return files, references, nil
}

// GraphAll returns the full indexed reference graph, plus every root file in
// the index (files that nothing else references) -- "graph all the indexed
// data".
func (ig *IndexedGraph) GraphAll() (references map[string][]string, roots []string, err error) {
	idx, _ := ig.indexer()

	references, err = idx.BuildReferenceGraph()
	if err != nil {
		return nil, nil, err
	}

	dg := NewDependencyGraph(ig.Fs, ig.Logger, references)
	referenced := dg.InvertDependencies()

	for node := range references {
		if _, isReferenced := referenced[node]; !isReferenced {
			roots = append(roots, node)
		}
	}
	sort.Strings(roots)
	return references, roots, nil
}

// ShowFile prints the reference tree rooted at path, followed by every other
// indexed file that shares at least one topic with it.
func (ig *IndexedGraph) ShowFile(path string) error {
	references, related, err := ig.GraphFile(path)
	if err != nil {
		return err
	}

	dg := NewDependencyGraph(ig.Fs, ig.Logger, references)
	fmt.Printf("References for %s:\n", path)
	dg.ListDependencyTree(path)

	fmt.Println("Related by topic:")
	if len(related) == 0 {
		fmt.Println("  (none)")
		return nil
	}
	for _, r := range related {
		fmt.Printf("  %s\n", r)
	}
	return nil
}

// ShowTopic prints every indexed file tagged with topic, then each of those
// files' own direct references, as a single tree rooted at the topic.
func (ig *IndexedGraph) ShowTopic(topic string) error {
	files, references, err := ig.GraphTopic(topic)
	if err != nil {
		return err
	}

	rootNode := "topic:" + topic
	merged := make(map[string][]string, len(references)+1)
	for k, v := range references {
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
	references, roots, err := ig.GraphAll()
	if err != nil {
		return err
	}

	dg := NewDependencyGraph(ig.Fs, ig.Logger, references)
	for _, root := range roots {
		fmt.Printf("References for %s:\n", root)
		dg.ListDependencyTree(root)
	}
	return nil
}
