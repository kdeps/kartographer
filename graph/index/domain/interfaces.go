package domain

// FileWalker walks a root directory and returns candidate file paths.
type FileWalker interface {
	Walk(root string, extensions []string) ([]string, error)
}

// ReferenceExtractor extracts explicit file references (links) from content.
type ReferenceExtractor interface {
	Extract(content string, fromPath string) []string
}

// TopicExtractor extracts declared topics/tags from content.
type TopicExtractor interface {
	Extract(content string, fromPath string) []string
}

// FileSetAware lets a ReferenceExtractor learn every path that will be
// indexed, for extractors that resolve non-relative references (e.g.
// import "pkg/foo") by name-matching against the indexed file set. A
// ReferenceExtractor that only resolves relative references does not need
// to implement this.
type FileSetAware interface {
	SetKnownFiles(paths []string)
}

// IndexStore persists indexed file records and maintains a topic inverted index.
type IndexStore interface {
	PutFile(rec *FileRecord) error
	GetFile(path string) (*FileRecord, bool, error)
	AllFiles() ([]*FileRecord, error)
	FilesByTopic(topic string) ([]string, error)
	Close() error
}
