package usecase

import (
	"github.com/spf13/afero"

	"github.com/kdeps/kartographer/graph/index/domain"
)

// IndexerService orchestrates walking a folder, extracting references and
// topics from each file, and persisting the results to an IndexStore.
type IndexerService struct {
	fs        afero.Fs
	walker    domain.FileWalker
	refExtr   domain.ReferenceExtractor
	topicExtr domain.TopicExtractor
	store     domain.IndexStore
}

func NewIndexerService(
	fs afero.Fs,
	walker domain.FileWalker,
	refExtr domain.ReferenceExtractor,
	topicExtr domain.TopicExtractor,
	store domain.IndexStore,
) *IndexerService {
	return &IndexerService{
		fs:        fs,
		walker:    walker,
		refExtr:   refExtr,
		topicExtr: topicExtr,
		store:     store,
	}
}

// IndexFolder walks root, extracting references and topics from every file
// matching extensions, and stores a FileRecord for each.
func (s *IndexerService) IndexFolder(root string, extensions []string) error {
	paths, err := s.walker.Walk(root, extensions)
	if err != nil {
		return err
	}

	for _, path := range paths {
		if err := s.indexFile(path); err != nil {
			return err
		}
	}
	return nil
}

func (s *IndexerService) indexFile(path string) error {
	data, err := afero.ReadFile(s.fs, path)
	if err != nil {
		return err
	}
	info, err := s.fs.Stat(path)
	if err != nil {
		return err
	}

	content := string(data)
	rec := domain.NewFileRecord(
		path,
		s.refExtr.Extract(content, path),
		s.topicExtr.Extract(content),
		info.ModTime().Unix(),
	)
	return s.store.PutFile(rec)
}

// BuildReferenceGraph returns a map of file path -> its explicit references,
// built from every indexed file. This is the "graph over all indexed data".
func (s *IndexerService) BuildReferenceGraph() (map[string][]string, error) {
	recs, err := s.store.AllFiles()
	if err != nil {
		return nil, err
	}

	graph := make(map[string][]string, len(recs))
	for _, rec := range recs {
		graph[rec.Path] = rec.References
	}
	return graph, nil
}

// FilesForTopic returns every indexed file tagged with topic.
func (s *IndexerService) FilesForTopic(topic string) ([]string, error) {
	return s.store.FilesByTopic(topic)
}

// TopicsForFile returns the topics declared by the given file.
func (s *IndexerService) TopicsForFile(path string) ([]string, error) {
	rec, found, err := s.store.GetFile(path)
	if err != nil || !found {
		return nil, err
	}
	return rec.Topics, nil
}

// FilesRelatedByTopic returns every other indexed file that shares at least
// one topic with the given file.
func (s *IndexerService) FilesRelatedByTopic(path string) ([]string, error) {
	topics, err := s.TopicsForFile(path)
	if err != nil || len(topics) == 0 {
		return nil, err
	}

	seen := map[string]bool{path: true}
	var related []string
	for _, topic := range topics {
		files, ferr := s.store.FilesByTopic(topic)
		if ferr != nil {
			return nil, ferr
		}
		for _, f := range files {
			if !seen[f] {
				seen[f] = true
				related = append(related, f)
			}
		}
	}
	return related, nil
}
