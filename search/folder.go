package search

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"unicode/utf8"
)

// folderBatch is how many documents IndexFolder writes at once.
const folderBatch = 2000

// DocOptions is how a file becomes a document. The caller owns the policy.
type DocOptions struct {
	Tokenizer *Tokenizer
	MaxSize   int64                             // larger files are left out; 0 keeps every size
	Preview   int                               // bytes of text kept as the preview, cut at a character
	Read      func(path string) (string, error) // a file's text; nil is ReadText
}

// DocID is the document id of the file at path.
func DocID(path string) string { return fmt.Sprintf("%x", path) }

// ReadDoc is the file at path, described by info, as a document with its
// tokens; ok is false when the file is too large or cannot be read.
func ReadDoc(path string, info fs.FileInfo, o DocOptions) (DocWithTokens, bool) {
	if o.MaxSize > 0 && info.Size() > o.MaxSize {
		return DocWithTokens{}, false
	}
	read := o.Read
	if read == nil {
		read = ReadText
	}
	text, err := read(path)
	if err != nil {
		return DocWithTokens{}, false
	}
	tokens := o.Tokenizer.Tokenize(text)
	words := make([]string, len(tokens))
	positions := make([]int, len(tokens))
	for i, t := range tokens {
		words[i], positions[i] = t.Text, t.Position
	}
	return DocWithTokens{
		Doc: &Document{
			ID: DocID(path), Path: path, ModTime: info.ModTime().Unix(), Size: info.Size(),
			Preview: cut(text, o.Preview),
		},
		Tokens: words, Positions: positions,
	}, true
}

// cut is the first n bytes of s, shortened to end on a whole character.
func cut(s string, n int) string {
	if n < 0 || len(s) <= n {
		return s
	}
	for n > 0 && !utf8.RuneStart(s[n]) {
		n--
	}
	return s[:n]
}

// FolderProgress is what IndexFolder has done so far.
type FolderProgress struct {
	Read      int // files read into the index
	Unchanged int // files skipped because their modification time is the indexed one
	Purged    int // documents removed because their file is gone (set once, at the end)
}

// FolderOptions is how IndexFolder reads a folder.
type FolderOptions struct {
	DocOptions
	Filter   *Filter
	Glob     string // file name pattern; "" matches every file
	Workers  int    // parallel readers; 0 or less uses GOMAXPROCS
	Progress func(FolderProgress)
	// ProgressEvery reports after this many files read, and ten times as many
	// unchanged; 0 reports only at the end.
	ProgressEvery int
}

// IndexFolder brings idx up to date with the files under root that pass the
// filter and the glob: a file whose modification time is the indexed one is
// skipped, the rest are read in parallel and written in batches, and the
// documents of files no longer there are removed. It returns how many files
// it read.
func IndexFolder(idx *InvertedIndex, root string, o FolderOptions) (int, error) {
	workers := o.Workers
	if workers <= 0 {
		workers = runtime.GOMAXPROCS(0)
	}
	var (
		p  FolderProgress
		mu sync.Mutex // p is counted by the writer and the walk; Progress sees one at a time
	)
	step := func(count func(*FolderProgress) bool) {
		mu.Lock()
		defer mu.Unlock()
		if count(&p) && o.Progress != nil {
			o.Progress(p)
		}
	}
	type job struct {
		path string
		info fs.FileInfo
	}
	jobs := make(chan job, folderBatch)
	docs := make(chan DocWithTokens, folderBatch)
	done := make(chan struct{})
	go func() {
		batch := make([]DocWithTokens, 0, folderBatch)
		for d := range docs {
			batch = append(batch, d)
			if len(batch) >= folderBatch {
				idx.BatchAddDocuments(batch)
				batch = batch[:0]
			}
			step(func(p *FolderProgress) bool {
				p.Read++
				return o.ProgressEvery > 0 && p.Read%o.ProgressEvery == 0
			})
		}
		idx.BatchAddDocuments(batch)
		close(done)
	}()
	var wg sync.WaitGroup
	for range workers {
		wg.Go(func() {
			for j := range jobs {
				if d, ok := ReadDoc(j.path, j.info, o.DocOptions); ok {
					docs <- d
				}
			}
		})
	}
	seen := map[string]bool{}
	walkErr := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil //nolint:nilerr // an unreadable entry is left out, not fatal
		}
		if d.IsDir() {
			if o.Filter.SkipEntry(d.Name()) {
				return fs.SkipDir
			}
			return nil
		}
		if o.Filter.SkipEntry(d.Name()) || !o.Filter.Indexable(d.Name()) || !matchGlob(o.Glob, d.Name()) {
			return nil
		}
		id := DocID(path)
		seen[id] = true
		info, err := d.Info()
		if err != nil {
			return nil //nolint:nilerr // a file that vanished mid-walk is left out
		}
		if info.ModTime().Unix() == idx.LookupModTime(id) {
			step(func(p *FolderProgress) bool {
				p.Unchanged++
				return o.ProgressEvery > 0 && p.Unchanged%(o.ProgressEvery*10) == 0 //nolint:mnd // unchanged files are cheap
			})
			return nil
		}
		jobs <- job{path: path, info: info}
		return nil
	})
	close(jobs)
	wg.Wait()
	close(docs)
	<-done
	purged := idx.PurgeDocs(seen)
	step(func(p *FolderProgress) bool { p.Purged = purged; return true })
	return p.Read, walkErr
}

func matchGlob(glob, name string) bool {
	if glob == "" {
		return true
	}
	ok, err := filepath.Match(glob, name)
	return err == nil && ok
}

// IndexFiles adds the files at paths to idx in one batch: each must be inside
// root, pass the filter and be readable. It returns how many it added.
func IndexFiles(idx *InvertedIndex, root string, paths []string, filter *Filter, o DocOptions) int {
	root = filepath.Clean(root)
	batch := make([]DocWithTokens, 0, len(paths))
	for _, path := range paths {
		if rel, err := filepath.Rel(root, path); err != nil || rel == ".." ||
			strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			continue
		}
		info, err := os.Stat(path)
		if err != nil || info.IsDir() || !filter.Indexable(filepath.Base(path)) {
			continue
		}
		if d, ok := ReadDoc(path, info, o); ok {
			batch = append(batch, d)
		}
	}
	idx.BatchAddDocuments(batch)
	return len(batch)
}

// OpenOrCreate opens the index at dbPath, creating its folder. A file there
// that is not an index is replaced by a new one. existed reports whether a
// valid index was already there.
func OpenOrCreate(dbPath string) (idx *InvertedIndex, existed bool, err error) {
	if err = os.MkdirAll(filepath.Dir(dbPath), 0o750); err != nil {
		return nil, false, err
	}
	_, statErr := os.Stat(dbPath)
	if idx, err = NewInvertedIndex(dbPath); err == nil {
		return idx, statErr == nil, nil
	}
	_ = os.Remove(dbPath)
	idx, err = NewInvertedIndex(dbPath)
	return idx, false, err
}
