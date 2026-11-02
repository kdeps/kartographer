package search

import (
	"regexp"
	"strings"
	"unicode"
)

// TokenizerOptions are the word rules of a tokenizer. The caller owns them.
type TokenizerOptions struct {
	MinWordLength int      // shorter words are dropped
	Stopwords     []string // words dropped, after lowercasing
	StemSuffixes  []string // tried in order; the first that fits is removed
	StemKeep      int      // a suffix is removed only when more than this many characters remain
}

// Token is a term with its position among the kept terms.
type Token struct {
	Text     string
	Position int
}

// Tokenizer splits text into lowercase, stemmed terms without stopwords.
type Tokenizer struct {
	o           TokenizerOptions
	stopWords   map[string]bool
	wordPattern *regexp.Regexp
}

// NewTokenizer returns a tokenizer that follows o.
func NewTokenizer(o TokenizerOptions) *Tokenizer {
	stop := make(map[string]bool, len(o.Stopwords))
	for _, w := range o.Stopwords {
		stop[w] = true
	}
	return &Tokenizer{o: o, stopWords: stop, wordPattern: regexp.MustCompile(`\w+`)}
}

// Tokenize splits text into tokens with positions.
func (t *Tokenizer) Tokenize(text string) []Token {
	var tokens []Token
	position := 0
	for _, match := range t.wordPattern.FindAllStringIndex(text, -1) {
		normalized := t.normalize(text[match[0]:match[1]])
		if normalized != "" && !t.stopWords[normalized] {
			tokens = append(tokens, Token{Text: normalized, Position: position})
			position++
		}
	}
	return tokens
}

// TokenizeToStrings returns the token texts without positions.
func (t *Tokenizer) TokenizeToStrings(text string) []string {
	tokens := t.Tokenize(text)
	out := make([]string, len(tokens))
	for i, tok := range tokens {
		out[i] = tok.Text
	}
	return out
}

// normalize lowercases a word, keeps its letters, digits and underscores, and stems it.
func (t *Tokenizer) normalize(word string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(word) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
			b.WriteRune(r)
		}
	}
	word = b.String()
	if len(word) < t.o.MinWordLength {
		return ""
	}
	return t.stem(word)
}

// stem removes the first suffix that leaves more than StemKeep characters.
func (t *Tokenizer) stem(word string) string {
	for _, suffix := range t.o.StemSuffixes {
		if len(word) > len(suffix)+t.o.StemKeep && strings.HasSuffix(word, suffix) {
			return word[:len(word)-len(suffix)]
		}
	}
	return word
}
