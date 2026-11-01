package lsp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadRegistryMapsExtensions(t *testing.T) {
	reg, err := LoadRegistry()
	require.NoError(t, err)

	lang, ok := reg.ForFile("src/app.py")
	require.True(t, ok)
	assert.Equal(t, "python", lang.ID)
	assert.Equal(t, []string{"pyright-langserver", "--stdio"}, lang.Servers[0])

	_, ok = reg.ForFile("README.md")
	assert.False(t, ok, "unregistered extensions have no language")
}

func TestServerIsNilWhenNotInstalled(t *testing.T) {
	t.Setenv("PATH", "")
	lang := Language{ID: "x", Servers: [][]string{{"definitely-not-installed"}}}
	assert.Nil(t, lang.Server())
}
