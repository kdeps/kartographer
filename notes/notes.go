// Package notes is a local store of durable notes about a module: decisions,
// constraints and gotchas. Notes are kept as one JSON file per module.
package notes

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"
)

// Note is one saved statement.
type Note struct {
	ID        string    `json:"id"`
	Text      string    `json:"text"`
	Tags      []string  `json:"tags,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// Store holds the notes of one module in a file.
type Store struct {
	path string
}

// Open returns the store kept at path. The file is created on first save.
func Open(path string) *Store {
	return &Store{path: path}
}

// DefaultPath is the notes file for a module, under the user config directory.
func DefaultPath(module string) (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	safe := strings.NewReplacer("/", "_", ":", "_").Replace(module)
	return filepath.Join(dir, "kartographer", "notes", safe+".json"), nil
}

// Add saves a note. An identical text already saved returns the existing note.
func (s *Store) Add(text string, tags []string) (Note, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return Note{}, errors.New("text is required")
	}
	all, err := s.load()
	if err != nil {
		return Note{}, err
	}
	for _, n := range all {
		if n.Text == text {
			return n, nil
		}
	}
	n := Note{ID: "n" + time.Now().UTC().Format("20060102150405.000000000"), Text: text, Tags: tags, CreatedAt: time.Now().UTC()}
	all = append(all, n)
	return n, s.save(all)
}

// Search returns up to k notes that share words with the query, best first.
func (s *Store) Search(query string, k int) ([]Note, error) {
	all, err := s.load()
	if err != nil {
		return nil, err
	}
	words := words(query)
	type scored struct {
		n     Note
		score int
	}
	var hits []scored
	for _, n := range all {
		text := strings.ToLower(n.Text)
		score := 0
		for _, w := range words {
			if strings.Contains(text, w) {
				score++
			}
		}
		if score > 0 {
			hits = append(hits, scored{n, score})
		}
	}
	sort.SliceStable(hits, func(i, j int) bool { return hits[i].score > hits[j].score })
	out := make([]Note, 0, len(hits))
	for _, h := range hits {
		out = append(out, h.n)
	}
	return capNotes(out, k), nil
}

// About returns up to k notes that mention target, a file path or a symbol name.
func (s *Store) About(target string, k int) ([]Note, error) {
	all, err := s.load()
	if err != nil {
		return nil, err
	}
	var out []Note
	for _, n := range all {
		if strings.Contains(n.Text, target) {
			out = append(out, n)
		}
	}
	return capNotes(out, k), nil
}

func (s *Store) load() ([]Note, error) {
	raw, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var all []Note
	return all, json.Unmarshal(raw, &all)
}

func (s *Store) save(all []Note) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(all, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, raw, 0o600)
}

func words(q string) []string {
	fields := strings.FieldsFunc(strings.ToLower(q), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_'
	})
	var out []string
	for _, f := range fields {
		if len(f) > 1 {
			out = append(out, f)
		}
	}
	return out
}

func capNotes(n []Note, k int) []Note {
	if k > 0 && len(n) > k {
		return n[:k]
	}
	return n
}
