package search

import (
	"reflect"
	"testing"
)

var testWords = TokenizerOptions{
	MinWordLength: 2,
	Stopwords:     []string{"the", "and", "over"},
	StemSuffixes:  []string{"ing", "ed", "ly", "es", "s"},
	StemKeep:      2,
}

func TestTokenizer(t *testing.T) {
	tk := NewTokenizer(testWords)
	got := tk.TokenizeToStrings("The quick brown fox jumps over the lazy dogs; testing")
	want := []string{"quick", "brown", "fox", "jump", "lazy", "dog", "test"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("tokens = %v, want %v", got, want)
	}
	if toks := tk.Tokenize("a I x"); len(toks) != 0 {
		t.Errorf("short words are dropped: %v", toks)
	}
	if toks := tk.Tokenize("red ding"); toks[0].Text != "red" || toks[1].Text != "ding" {
		t.Errorf("a stem must keep more than StemKeep characters: %v", toks)
	}
	if toks := tk.Tokenize("alpha beta"); toks[1].Position != 1 {
		t.Errorf("positions count kept terms: %v", toks)
	}
}
