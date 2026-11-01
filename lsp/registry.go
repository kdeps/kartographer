package lsp

import (
	_ "embed"
	"os/exec"
	"path/filepath"
	"slices"

	"gopkg.in/yaml.v3"
)

//go:embed registry.yaml
var registryYAML []byte

// Language is one entry of registry.yaml.
type Language struct {
	ID         string     `yaml:"id"`
	Extensions []string   `yaml:"extensions"`
	Servers    [][]string `yaml:"servers"`
}

// Registry is the parsed registry.yaml.
type Registry struct {
	Languages []Language `yaml:"languages"`
}

// LoadRegistry parses the embedded registry.yaml.
func LoadRegistry() (Registry, error) {
	var r Registry
	err := yaml.Unmarshal(registryYAML, &r)
	return r, err
}

// ForFile is the language of a file, by extension, or false.
func (r Registry) ForFile(path string) (Language, bool) {
	ext := filepath.Ext(path)
	for _, l := range r.Languages {
		if slices.Contains(l.Extensions, ext) {
			return l, true
		}
	}
	return Language{}, false
}

// Server is the first of the language's servers installed on PATH, or nil.
func (l Language) Server() []string {
	for _, s := range l.Servers {
		if _, err := exec.LookPath(s[0]); err == nil {
			return s
		}
	}
	return nil
}
