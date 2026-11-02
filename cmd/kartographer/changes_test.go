package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrintChangesListsChangedAndImpact(t *testing.T) {
	configureRules(t)
	ix, err := load(fixture, true)
	require.NoError(t, err)

	var out bytes.Buffer
	printChanges(&out, ix, "+++ b/pkg/a.go\n@@ -5 +5 @@\n", 2, 60)
	text := out.String()
	assert.True(t, strings.HasPrefix(text, "changed:\n  example.com/fx/pkg.helper  pkg/a.go:5\nimpact:\n"), text)
	assert.Contains(t, text, "example.com/fx/pkg.Entry")
}

func TestPrintChangesSaysNoneWhenNothingMatches(t *testing.T) {
	configureRules(t)
	ix, err := load(fixture, true)
	require.NoError(t, err)

	var out bytes.Buffer
	printChanges(&out, ix, "", 2, 60)
	assert.Contains(t, out.String(), "changed:\n  none\n")
}

func TestChangesCommandReportsAGitFailure(t *testing.T) {
	configureRules(t)
	var out bytes.Buffer
	err := changesCommand(&out, t.TempDir(), "HEAD", 2, 60, true)
	assert.ErrorContains(t, err, "git diff HEAD")
}
