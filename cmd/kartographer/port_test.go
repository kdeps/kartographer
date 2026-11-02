package main

import (
	"bytes"
	"encoding/xml"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func gitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".git", "hooks"), 0o755))
	return dir
}

func TestHookInstallStatusUninstall(t *testing.T) {
	dir := gitRepo(t)
	var out bytes.Buffer
	require.NoError(t, hookCommand(&out, dir, "install"))
	post, err := os.ReadFile(filepath.Join(dir, ".git", "hooks", "post-commit"))
	require.NoError(t, err)
	assert.Contains(t, string(post), hookBegin)
	assert.Contains(t, string(post), "KARTOGRAPHER_RULES")

	out.Reset()
	require.NoError(t, hookCommand(&out, dir, "status"))
	assert.Contains(t, out.String(), "post-commit: installed")
	assert.Contains(t, out.String(), "post-checkout: installed")

	out.Reset()
	require.NoError(t, hookCommand(&out, dir, "uninstall"))
	_, err = os.Stat(filepath.Join(dir, ".git", "hooks", "post-commit"))
	assert.True(t, os.IsNotExist(err), "a hook left with only its shebang is removed")
}

func TestHookKeepsExistingHookContent(t *testing.T) {
	dir := gitRepo(t)
	hook := filepath.Join(dir, ".git", "hooks", "post-commit")
	require.NoError(t, os.WriteFile(hook, []byte("#!/bin/sh\necho mine\n"), 0o755))

	var out bytes.Buffer
	require.NoError(t, hookCommand(&out, dir, "install"))
	raw, _ := os.ReadFile(hook)
	assert.Contains(t, string(raw), "echo mine")
	assert.Contains(t, string(raw), hookBegin)

	require.NoError(t, hookCommand(&out, dir, "uninstall"))
	raw, err := os.ReadFile(hook)
	require.NoError(t, err)
	assert.Contains(t, string(raw), "echo mine")
	assert.NotContains(t, string(raw), hookBegin)
}

func TestHookErrors(t *testing.T) {
	var out bytes.Buffer
	assert.Error(t, hookCommand(&out, t.TempDir(), "install"), "not a git repository")

	dir := gitRepo(t)
	err := hookCommand(&out, dir, "frob")
	assert.ErrorContains(t, err, "hook needs")
}

func TestHookInstallIsIdempotent(t *testing.T) {
	dir := gitRepo(t)
	var out bytes.Buffer
	require.NoError(t, hookCommand(&out, dir, "install"))
	require.NoError(t, hookCommand(&out, dir, "install"))
	raw, _ := os.ReadFile(filepath.Join(dir, ".git", "hooks", "post-commit"))
	assert.Equal(t, 1, strings.Count(string(raw), hookBegin), "installing twice keeps one block")
}

func TestWatchIndexesThenRebuildsOnlyOnChange(t *testing.T) {
	var out bytes.Buffer
	require.NoError(t, watchCommand(&out, fixture, 0, 3, true))
	assert.Equal(t, 1, strings.Count(out.String(), "indexed"), "no change after the first build means no rebuild")
	assert.Contains(t, out.String(), "indexed example.com/fx: 5 symbols")
}

func TestWatchRebuildsAfterAChange(t *testing.T) {
	src := t.TempDir()
	require.NoError(t, copyTree(fixture, src))
	var out bytes.Buffer
	// First round builds; then the file changes before the second round.
	done := make(chan error, 1)
	go func() { done <- watchCommand(&out, src, 200*time.Millisecond, 2, true) }()
	time.Sleep(50 * time.Millisecond)
	f := filepath.Join(src, "pkg", "a.go")
	raw, err := os.ReadFile(f)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(f, append(raw, []byte("\nfunc more() int { return 2 }\n")...), 0o600))
	require.NoError(t, <-done)
	assert.Equal(t, 2, strings.Count(out.String(), "indexed"))
}

func TestExportGraphMLIsWellFormed(t *testing.T) {
	configureRules(t)
	var out bytes.Buffer
	require.NoError(t, exportCommand(&out, "graphml", "", fixture, true))
	assert.Contains(t, out.String(), `<node id="example.com/fx/pkg.leaf">`)
	assert.Contains(t, out.String(), `source="example.com/fx/pkg.helper" target="example.com/fx/pkg.leaf"`)

	var tokens int
	dec := xml.NewDecoder(strings.NewReader(out.String()))
	for {
		_, err := dec.Token()
		if err != nil {
			break
		}
		tokens++
	}
	assert.Greater(t, tokens, 10, "the document decodes")
}

func TestExportWritesFileAndRejectsOtherFormats(t *testing.T) {
	configureRules(t)
	path := filepath.Join(t.TempDir(), "g.graphml")
	var out bytes.Buffer
	require.NoError(t, exportCommand(&out, "graphml", path, fixture, true))
	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(raw), "<graphml")
	assert.Contains(t, out.String(), "wrote "+path)

	assert.ErrorContains(t, exportCommand(&out, "pdf", "", fixture, true), "export supports")
}

func TestPortedCommandsThroughRun(t *testing.T) {
	_, errw, code := runCLI(t, "export", "graphml", "-dir", fixture, "-no-cache")
	assert.Equal(t, 0, code, errw)

	_, errw, code = runCLI(t, "export", "-dir", fixture, "-no-cache")
	assert.Equal(t, 2, code)
	assert.Contains(t, errw, "export needs a format")

	_, errw, code = runCLI(t, "hook", "status", "-dir", t.TempDir())
	assert.Equal(t, 1, code)
	assert.Contains(t, errw, "not a git repository")
}

func copyTree(src, dst string) error {
	return filepath.Walk(src, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, p)
		if info.IsDir() {
			return os.MkdirAll(filepath.Join(dst, rel), 0o755)
		}
		raw, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(dst, rel), raw, 0o600)
	})
}
