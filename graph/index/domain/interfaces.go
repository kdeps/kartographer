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
	Extract(content string) []string
}

// IndexStore persists indexed file records and maintains a topic inverted index.
type IndexStore interface {
	PutFile(rec *FileRecord) error
	GetFile(path string) (*FileRecord, bool, error)
	AllFiles() ([]*FileRecord, error)
	FilesByTopic(topic string) ([]string, error)
	Close() error
}
