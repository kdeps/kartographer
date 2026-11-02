// Package text splits free text into the words a keyword index matches on.
package text

import (
	"regexp"
	"strings"
)

var wordRe = regexp.MustCompile(`[\p{L}\p{N}]+`) //nolint:gochecknoglobals // fixed pattern

// Options is the policy for turning text into words. The caller owns it: a
// word shorter than MinWordLength or in Stopwords is dropped, and a word is
// stemmed when it is at least MinStemLength once its suffix is removed.
type Options struct {
	MinWordLength int
	MinStemLength int
	Stopwords     map[string]bool
}

// Words returns the stemmed words of s that pass o. camelCase and snake_case
// are split first, and the original text is kept too, so both forms match.
func Words(s string, o Options) []string {
	var out []string
	for _, w := range wordRe.FindAllString(strings.ToLower(SplitCamel(s)), -1) {
		if len(w) >= o.MinWordLength && !o.Stopwords[w] {
			out = append(out, Stem(w, o.MinStemLength))
		}
	}
	return out
}

// Stem trims a plural or verb ending so word forms meet: billing and bill,
// invoices and invoice, deployed and deploy. A double s (process) stays.
func Stem(w string, minStem int) string {
	for _, suffix := range []string{"ing", "ed", "s"} {
		if !strings.HasSuffix(w, suffix) || len(w)-len(suffix) < minStem {
			continue
		}
		if suffix == "s" && strings.HasSuffix(w, "ss") {
			return w
		}
		return strings.TrimSuffix(w, suffix)
	}
	return w
}

// SplitCamel returns s followed by a copy with camelCase and separator
// characters (_ - / .) turned into spaces.
func SplitCamel(s string) string {
	var b strings.Builder
	prevLower := false
	for _, r := range s {
		isUpper := r >= 'A' && r <= 'Z'
		if isUpper && prevLower {
			b.WriteByte(' ')
		}
		if r == '_' || r == '-' || r == '/' || r == '.' {
			r = ' '
		}
		b.WriteRune(r)
		prevLower = r >= 'a' && r <= 'z'
	}
	return s + " " + b.String()
}

// Set is the set of ws.
func Set(ws []string) map[string]bool {
	m := make(map[string]bool, len(ws))
	for _, w := range ws {
		m[w] = true
	}
	return m
}
