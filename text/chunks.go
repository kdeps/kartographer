package text

import "strings"

// Chunks splits text into pieces of at most max bytes, cutting at the last
// paragraph break past half of max when there is one. Each piece is trimmed;
// empty text has no pieces.
func Chunks(text string, max int) []string {
	text = strings.TrimSpace(text)
	var out []string
	for text != "" {
		cut := len(text)
		if max > 0 && cut > max {
			cut = max
			if i := strings.LastIndex(text[:cut], "\n\n"); i > max/2 {
				cut = i
			}
		}
		out = append(out, text[:cut])
		text = strings.TrimSpace(text[cut:])
	}
	return out
}
