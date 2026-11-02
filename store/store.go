// Package store reads and writes the JSON state files of a module's graph and
// notes. A write replaces the file in one step, so a reader never sees half of it.
package store

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// WriteJSON saves v as indented JSON at file, creating its directory. The file
// is replaced atomically: the data is written beside it and renamed over it.
func WriteJSON(file string, v any) error {
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil { //nolint:mnd // directory permissions
		return err
	}
	data, err := json.MarshalIndent(v, "", " ")
	if err != nil {
		return err
	}
	tmp := file + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, file)
}

// ReadJSON loads the JSON file at file into v. It returns the read error, so a
// missing file can be told apart with os.IsNotExist or errors.Is.
func ReadJSON(file string, v any) error {
	data, err := os.ReadFile(file)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}
