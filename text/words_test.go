package text

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var opts = Options{MinWordLength: 3, MinStemLength: 3, Stopwords: map[string]bool{"the": true}} //nolint:gochecknoglobals // test fixture

func TestWordsDropShortAndStopwords(t *testing.T) {
	got := Words("The billing of an invoice", opts)
	assert.Contains(t, got, "bill")
	assert.Contains(t, got, "invoice")
	assert.NotContains(t, got, "the")
}

func TestWordsSplitCamelAndSnake(t *testing.T) {
	got := Words("BillingService.charge_card", opts)
	assert.Contains(t, got, "billingservice", "the whole name is kept")
	assert.Contains(t, got, "bill")
	assert.Contains(t, got, "service")
	assert.Contains(t, got, "charge")
	assert.Contains(t, got, "card")
}

func TestStem(t *testing.T) {
	assert.Equal(t, "bill", Stem("billing", 3))
	assert.Equal(t, "deploy", Stem("deployed", 3))
	assert.Equal(t, "invoice", Stem("invoices", 3))
	assert.Equal(t, "process", Stem("process", 3), "a double s stays")
	assert.Equal(t, "is", Stem("is", 3), "too short to stem")
}

func TestSplitCamelKeepsOriginal(t *testing.T) {
	assert.Equal(t, "fooBar foo Bar", SplitCamel("fooBar"))
}

func TestSet(t *testing.T) {
	assert.Equal(t, map[string]bool{"a": true, "b": true}, Set([]string{"a", "b", "a"}))
}
