package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const fixture = "../../codegraph/testdata/fixture"

// rulesFile writes the minimum rules the commands need.
func rulesFile(t *testing.T) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "rules.yaml")
	require.NoError(t, os.WriteFile(p, []byte("ranking:\n  name_weight: 3\n  path_weight: 1\n  degree_weight: 0.1\n  body_lines: 30\n  stopwords: [the, of, is]\npaths:\n  max_depth: 8\n  max_paths: 10\nlimits:\n  max_result: 300\n"), 0o600))
	return p
}

func runCLI(t *testing.T, args ...string) (string, string, int) {
	t.Helper()
	var out, errw bytes.Buffer
	code := run(append([]string{"-rules", rulesFile(t)}, args...), &out, &errw)
	return out.String(), errw.String(), code
}

func TestUpdateReportsTheIndex(t *testing.T) {
	out, _, code := runCLI(t, "update", "-no-cache", fixture)
	assert.Equal(t, 0, code)
	assert.Contains(t, out, "indexed example.com/fx: 5 symbols, 4 call edges")
}

func TestQueryFindsSymbolsWithAndWithoutSource(t *testing.T) {
	out, _, code := runCLI(t, "query", "leaf", "-dir", fixture, "-no-cache", "-skeleton")
	assert.Equal(t, 0, code)
	assert.Contains(t, out, "example.com/fx/pkg.leaf  pkg/a.go:7")
	assert.NotContains(t, out, "func leaf")

	out, _, _ = runCLI(t, "query", "leaf", "-dir", fixture, "-no-cache")
	assert.Contains(t, out, "func leaf")

	out, _, _ = runCLI(t, "query", "zzz-nothing", "-dir", fixture, "-no-cache")
	assert.Equal(t, "no matches\n", out)
}

func TestPathUndirectedAndDirected(t *testing.T) {
	out, _, _ := runCLI(t, "path", "leaf", "TestEntry", "-dir", fixture, "-no-cache")
	assert.Equal(t, "example.com/fx/pkg.leaf -- example.com/fx/pkg.helper -- example.com/fx/pkg.Entry -- example.com/fx/pkg.TestEntry\n", out)

	out, _, _ = runCLI(t, "path", "Entry", "leaf", "--directed", "-dir", fixture, "-no-cache")
	assert.Equal(t, "example.com/fx/pkg.Entry -> example.com/fx/pkg.helper -> example.com/fx/pkg.leaf\n", out)

	out, _, _ = runCLI(t, "path", "leaf", "Entry", "--directed", "-dir", fixture, "-no-cache")
	assert.Equal(t, "no path\n", out)
}

func TestExplainShowsSourceAndNeighbours(t *testing.T) {
	out, _, code := runCLI(t, "explain", "helper", "-dir", fixture, "-no-cache")
	assert.Equal(t, 0, code)
	assert.Contains(t, out, "example.com/fx/pkg.helper  pkg/a.go:5")
	assert.Contains(t, out, "neighbours:")
	assert.Contains(t, out, "example.com/fx/pkg.leaf")
}

func TestAffectedListsNearestFirst(t *testing.T) {
	out, _, code := runCLI(t, "affected", "leaf", "-dir", fixture, "-no-cache")
	assert.Equal(t, 0, code)
	lines := strings.Split(strings.TrimSpace(out), "\n")
	require.Len(t, lines, 3)
	assert.Contains(t, lines[2], "example.com/fx/pkg.Entry", "two hops away comes last")

	out, _, _ = runCLI(t, "affected", "leaf", "-dir", fixture, "-no-cache", "-depth", "1")
	assert.NotContains(t, out, "Entry")
}

func TestUsageAndErrors(t *testing.T) {
	var out, errBuf bytes.Buffer
	assert.Equal(t, 2, run([]string{"-rules", rulesFile(t)}, &out, &errBuf), "no command prints usage")
	assert.Contains(t, errBuf.String(), "usage:")

	var o2, e2 bytes.Buffer
	assert.Equal(t, 2, run([]string{"update"}, &o2, &e2), "missing -rules is a usage error")
	assert.Contains(t, e2.String(), "-rules is required")

	_, errw2, code2 := runCLI(t, "path", "only-one", "-dir", fixture, "-no-cache")
	assert.Equal(t, 2, code2)
	assert.Contains(t, errw2, "path needs")

	_, errw3, code3 := runCLI(t, "frobnicate", "-no-cache")
	assert.Equal(t, 2, code3)
	assert.Contains(t, errw3, "unknown command")

	_, errw4, code4 := runCLI(t, "explain", "nope", "-dir", fixture, "-no-cache")
	assert.Equal(t, 1, code4)
	assert.Contains(t, errw4, "no symbol matches")
}

func TestBadRulesFileFails(t *testing.T) {
	bad := filepath.Join(t.TempDir(), "bad.yaml")
	require.NoError(t, os.WriteFile(bad, []byte("ranking: [unclosed"), 0o600))
	var out, errw bytes.Buffer
	assert.Equal(t, 1, run([]string{"-rules", bad, "update", fixture}, &out, &errw))
	assert.Contains(t, errw.String(), "rules:")
}

func TestSplitArgsKeepsFlagValues(t *testing.T) {
	pos, flags := splitArgs([]string{"leaf", "-dir", "x", "-skeleton", "TestEntry", "-k", "3"})
	assert.Equal(t, []string{"leaf", "TestEntry"}, pos)
	assert.Equal(t, []string{"-dir", "x", "-skeleton", "-k", "3"}, flags)
}
