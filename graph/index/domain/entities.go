package domain

// FileRecord is the indexed representation of a single file.
type FileRecord struct {
	Path       string
	References []string
	Topics     []string
	ModTime    int64
}

func NewFileRecord(path string, references, topics []string, modTime int64) *FileRecord {
	return &FileRecord{
		Path:       path,
		References: references,
		Topics:     topics,
		ModTime:    modTime,
	}
}
