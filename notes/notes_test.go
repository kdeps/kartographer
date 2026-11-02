package notes

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddDeduplicatesAndPersists(t *testing.T) {
	path := filepath.Join(t.TempDir(), "n.json")
	s := Open(path)
	first, err := s.Add("  Charge retries are disabled on purpose.  ", []string{"billing"})
	require.NoError(t, err)
	again, err := s.Add("Charge retries are disabled on purpose.", nil)
	require.NoError(t, err)
	assert.Equal(t, first.ID, again.ID, "identical text returns the saved note")

	reopened := Open(path)
	got, err := reopened.Search("retries", 10)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "Charge retries are disabled on purpose.", got[0].Text)
}

func TestAddRejectsEmpty(t *testing.T) {
	_, err := Open(filepath.Join(t.TempDir(), "n.json")).Add("   ", nil)
	assert.ErrorContains(t, err, "text is required")
}

func TestSearchRanksByWordOverlap(t *testing.T) {
	s := Open(filepath.Join(t.TempDir(), "n.json"))
	_, _ = s.Add("retry policy for charges", nil)
	_, _ = s.Add("charges are idempotent", nil)
	got, err := s.Search("charges retry", 0)
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Contains(t, got[0].Text, "retry", "two matching words beat one")

	none, err := s.Search("zzz", 5)
	require.NoError(t, err)
	assert.Empty(t, none)
}

func TestAboutMatchesTargetsAndCaps(t *testing.T) {
	s := Open(filepath.Join(t.TempDir(), "n.json"))
	_, _ = s.Add("BillingService.charge must stay idempotent", nil)
	_, _ = s.Add("see pkg/billing/charge.go for the retry", nil)
	_, _ = s.Add("unrelated note", nil)

	got, err := s.About("BillingService.charge", 5)
	require.NoError(t, err)
	require.Len(t, got, 1)

	capped, err := s.About("charge", 1)
	require.NoError(t, err)
	assert.Len(t, capped, 1)
}

func TestDefaultPathIsPerModule(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	p, err := DefaultPath("example.com/fx")
	require.NoError(t, err)
	assert.Contains(t, p, "example.com_fx.json")
}

func TestLoadMissingFileIsEmpty(t *testing.T) {
	got, err := Open(filepath.Join(t.TempDir(), "absent.json")).Search("x", 1)
	require.NoError(t, err)
	assert.Empty(t, got)
}
