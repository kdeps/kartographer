package notes

import (
	"errors"
	"fmt"
	"strings"
)

const frontDelim = "---"

// EditFrontmatter sets top-level keys in a note's frontmatter, keeping the
// other lines. A key in set replaces its line; a key in order that the block
// lacks is appended in order. A note with no frontmatter gets a block. Values
// are written as YAML double-quoted strings when they need it.
func EditFrontmatter(text string, set map[string]string, order []string) (string, error) {
	var head []string
	body := text
	if strings.HasPrefix(text, frontDelim+"\n") {
		rest := text[len(frontDelim)+1:]
		end := strings.Index("\n"+rest, "\n"+frontDelim+"\n")
		if end < 0 {
			if !strings.HasSuffix(rest, "\n"+frontDelim) {
				return "", errors.New("unterminated frontmatter")
			}
			end = len(rest) - len(frontDelim)
		}
		if end > 0 {
			head = strings.Split(strings.TrimSuffix(rest[:end], "\n"), "\n")
		}
		body = strings.TrimPrefix(rest[end:], frontDelim)
		body = strings.TrimPrefix(body, "\n")
	}
	done := map[string]bool{}
	for i, line := range head {
		key, _, ok := strings.Cut(line, ":")
		if !ok || strings.HasPrefix(line, " ") {
			continue
		}
		if v, want := set[key]; want {
			head[i] = key + ": " + Scalar(v)
			done[key] = true
		}
	}
	for _, key := range order {
		if v, want := set[key]; want && !done[key] {
			head = append(head, key+": "+Scalar(v))
		}
	}
	return frontDelim + "\n" + strings.Join(head, "\n") + "\n" + frontDelim + "\n" + body, nil
}

// Scalar returns s as a YAML scalar: plain when that is safe, otherwise quoted.
func Scalar(s string) string {
	if s == "" || strings.ContainsAny(s, ":#{}[],&*!|>'\"%@`\n") || strings.TrimSpace(s) != s {
		return fmt.Sprintf("%q", s)
	}
	return s
}
