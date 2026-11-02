package notes

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEditFrontmatterReplacesAndAppends(t *testing.T) {
	text := "---\ntitle: Old\ntags: [a]\n---\nbody line\n"
	got, err := EditFrontmatter(text, map[string]string{"title": "New", "status": "done"}, []string{"title", "status"})
	require.NoError(t, err)
	assert.Equal(t, "---\ntitle: New\ntags: [a]\nstatus: done\n---\nbody line\n", got)
}

func TestEditFrontmatterCreatesTheBlock(t *testing.T) {
	got, err := EditFrontmatter("just a body\n", map[string]string{"status": "open"}, []string{"status"})
	require.NoError(t, err)
	assert.Equal(t, "---\nstatus: open\n---\njust a body\n", got)
}

func TestEditFrontmatterRefusesAnUnterminatedBlock(t *testing.T) {
	_, err := EditFrontmatter("---\ntitle: x\n", map[string]string{"title": "y"}, nil)
	assert.ErrorContains(t, err, "unterminated frontmatter")
}

func TestScalarQuotesOnlyWhenNeeded(t *testing.T) {
	assert.Equal(t, "plain", Scalar("plain"))
	assert.Equal(t, `"a: b"`, Scalar("a: b"))
	assert.Equal(t, `""`, Scalar(""))
	assert.Equal(t, `" padded "`, Scalar(" padded "))
}
