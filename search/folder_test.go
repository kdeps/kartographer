package search

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func folderOptions() FolderOptions {
	return FolderOptions{
		DocOptions: DocOptions{Tokenizer: NewTokenizer(TokenizerOptions{MinWordLength: 2}), Preview: 5},
		Filter: NewFilter(FilterOptions{
			SkipHidden: true, SkipDirs: []string{"vendor"}, Extensions: []string{".md"}, SecretNames: []string{".env"},
		}),
		Workers: 2,
	}
}

func write(t *testing.T, path, text string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o750))
	require.NoError(t, os.WriteFile(path, []byte(text), 0o600))
}

func TestIndexFolder(t *testing.T) {
	root := t.TempDir()
	write(t, filepath.Join(root, "a.md"), "zorbulator reconciles invoices")
	write(t, filepath.Join(root, "b.txt"), "zorbulator text, wrong extension")
	write(t, filepath.Join(root, "vendor", "c.md"), "zorbulator vendored")
	write(t, filepath.Join(root, ".hidden", "d.md"), "zorbulator hidden")
	write(t, filepath.Join(root, ".env"), "zorbulator=secret")
	idx, existed, err := OpenOrCreate(filepath.Join(t.TempDir(), "x", "index.db"))
	require.NoError(t, err)
	assert.False(t, existed)
	defer func() { _ = idx.Close() }()

	var last FolderProgress
	o := folderOptions()
	o.Progress = func(p FolderProgress) { last = p }
	n, err := IndexFolder(idx, root, o)
	require.NoError(t, err)
	assert.Equal(t, 1, n, "only a.md passes the filter")
	res := idx.Search([]string{"zorbulator"})
	require.Len(t, res, 1)
	assert.Equal(t, filepath.Join(root, "a.md"), res[0].Document.Path)
	assert.Equal(t, "zorbu", res[0].Document.Preview, "preview cut to 5 bytes")

	n, err = IndexFolder(idx, root, o)
	require.NoError(t, err)
	assert.Zero(t, n, "an unchanged file is not read again")
	assert.Equal(t, 1, last.Unchanged)

	require.NoError(t, os.Remove(filepath.Join(root, "a.md")))
	_, err = IndexFolder(idx, root, o)
	require.NoError(t, err)
	assert.Equal(t, 1, last.Purged, "a removed file's document is purged")
	assert.Empty(t, idx.Search([]string{"zorbulator"}))
}

func TestIndexFiles(t *testing.T) {
	root := t.TempDir()
	in, out := filepath.Join(root, "in.md"), filepath.Join(t.TempDir(), "out.md")
	write(t, in, "alpha beta")
	write(t, out, "alpha gamma")
	idx, _, err := OpenOrCreate(filepath.Join(t.TempDir(), "index.db"))
	require.NoError(t, err)
	defer func() { _ = idx.Close() }()
	o := folderOptions()
	assert.Equal(t, 1, IndexFiles(idx, root, []string{in, out, filepath.Join(root, "missing.md")}, o.Filter, o.DocOptions))
	assert.Len(t, idx.Search([]string{"alpha"}), 1, "a file outside root is left out")
}

func TestReadDoc(t *testing.T) {
	p := filepath.Join(t.TempDir(), "a.md")
	write(t, p, "héllo wörld")
	info, err := os.Stat(p)
	require.NoError(t, err)
	o := DocOptions{Tokenizer: NewTokenizer(TokenizerOptions{MinWordLength: 2}), Preview: 2}
	d, ok := ReadDoc(p, info, o)
	require.True(t, ok)
	assert.Equal(t, "h", d.Doc.Preview, "the cut never splits a character")
	assert.Equal(t, DocID(p), d.Doc.ID)
	o.MaxSize = 3
	_, ok = ReadDoc(p, info, o)
	assert.False(t, ok, "too large")
	o.MaxSize, o.Read = 0, func(string) (string, error) { return "from reader", nil }
	d, ok = ReadDoc(p, info, o)
	require.True(t, ok)
	assert.True(t, strings.HasPrefix(d.Doc.Preview, "fr"))
}

func TestOpenOrCreate_ReplacesANonIndex(t *testing.T) {
	p := filepath.Join(t.TempDir(), "index.db")
	write(t, p, "not a database")
	idx, existed, err := OpenOrCreate(p)
	require.NoError(t, err)
	assert.False(t, existed)
	require.NoError(t, idx.Close())
	idx, existed, err = OpenOrCreate(p)
	require.NoError(t, err)
	assert.True(t, existed)
	require.NoError(t, idx.Close())
}

func TestIndexFolder_ProgressWhileReadingAndSkipping(t *testing.T) {
	root := t.TempDir()
	for i := range 40 {
		write(t, filepath.Join(root, "f"+strings.Repeat("x", i)+".md"), "word")
	}
	idx, _, err := OpenOrCreate(filepath.Join(t.TempDir(), "index.db"))
	require.NoError(t, err)
	defer func() { _ = idx.Close() }()
	o := folderOptions()
	o.ProgressEvery = 1
	var calls int
	o.Progress = func(FolderProgress) { calls++ }
	_, err = IndexFolder(idx, root, o)
	require.NoError(t, err)
	write(t, filepath.Join(root, "new.md"), "word")
	_, err = IndexFolder(idx, root, o)
	require.NoError(t, err)
	assert.Positive(t, calls, "progress is reported; -race checks it is not raced")
}
