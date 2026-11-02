package notes

import (
	"bytes"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// FrontmatterKeys names the front matter keys a parse reads tags and links from.
// The caller owns the names.
type FrontmatterKeys struct {
	Tags  string
	Links string
}

// Markdown is one parsed markdown note: its front matter and body, its tags,
// the targets of its links, and the 1-based line each link and tag was first
// seen on. Links inside fenced or inline code are not links.
type Markdown struct {
	Meta      map[string]any
	Body      string
	Tags      []string
	Links     []string // wiki link targets in the body
	MetaLinks []string // targets of the front matter links key
	MDLinks   []string // relative markdown note link targets in the body
	LinkLine  map[string]int
	MetaLine  map[string]int
	MDLine    map[string]int
	TagLine   map[string]int
}

// ParseMarkdown parses a note's front matter, body, tags and links.
func ParseMarkdown(data []byte, k FrontmatterKeys) Markdown {
	meta, body := SplitFrontmatter(data)
	m := Markdown{Meta: meta, Body: body, Tags: MetaList(meta, k.Tags)}
	for _, l := range MetaList(meta, k.Links) {
		if t := WikiLinkTargets(l); len(t) > 0 {
			m.MetaLinks = append(m.MetaLinks, t...)
		} else if l = strings.TrimSpace(l); l != "" {
			m.MetaLinks = append(m.MetaLinks, l)
		}
	}
	prose := StripCode(body)
	m.Links = append(m.Links, WikiLinkTargets(prose)...)
	m.MDLinks = append(m.MDLinks, MarkdownNoteLinks(prose)...)
	m.indexLines(data)
	return m
}

// indexLines records the file line of each link and tag, skipping fenced code
// the way the link scan does.
func (m *Markdown) indexLines(data []byte) {
	m.LinkLine = map[string]int{}
	m.MetaLine = map[string]int{}
	m.MDLine = map[string]int{}
	m.TagLine = map[string]int{}
	inFM := false
	seenFM := 0
	fence := ""
	for i, line := range strings.Split(string(data), "\n") {
		num := i + 1
		t := strings.TrimSpace(line)
		if seenFM < 2 && t == frontDelim {
			if !inFM {
				inFM = true
				seenFM = 1
			} else {
				inFM = false
				seenFM = 2
			}
			continue
		}
		if inFM {
			m.indexFrontmatterLine(line, num)
			continue
		}
		if fence == "" && len(t) >= 3 && (strings.HasPrefix(t, "```") || strings.HasPrefix(t, "~~~")) {
			fence = t[:3]
			continue
		}
		if fence != "" {
			if strings.HasPrefix(t, fence) {
				fence = ""
			}
			continue
		}
		m.indexBodyLine(StripInlineCode(line), num)
	}
}

func (m *Markdown) indexFrontmatterLine(line string, num int) {
	for _, target := range WikiLinkTargets(line) {
		if _, ok := m.MetaLine[target]; !ok {
			m.MetaLine[target] = num
		}
	}
	low := strings.ToLower(line)
	for _, tag := range m.Tags {
		if m.TagLine[tag] == 0 && strings.Contains(low, strings.ToLower(tag)) {
			m.TagLine[tag] = num
		}
	}
}

func (m *Markdown) indexBodyLine(line string, num int) {
	for _, target := range WikiLinkTargets(line) {
		if _, ok := m.LinkLine[target]; !ok {
			m.LinkLine[target] = num
		}
	}
	for _, target := range MarkdownNoteLinks(line) {
		if _, ok := m.MDLine[target]; !ok {
			m.MDLine[target] = num
		}
	}
}

// StripCode blanks fenced code blocks and inline code spans: a link inside an
// example is not a link.
func StripCode(body string) string {
	var b strings.Builder
	fence := ""
	for _, line := range strings.SplitAfter(body, "\n") {
		t := strings.TrimSpace(line)
		if fence == "" && (strings.HasPrefix(t, "```") || strings.HasPrefix(t, "~~~")) {
			fence = t[:3]
			continue
		}
		if fence != "" {
			if strings.HasPrefix(t, fence) {
				fence = ""
			}
			continue
		}
		b.WriteString(StripInlineCode(line))
	}
	return b.String()
}

// SplitFrontmatter separates a leading "---" YAML block from the body. A note
// without one, or with invalid YAML, has empty metadata.
func SplitFrontmatter(data []byte) (map[string]any, string) {
	text := string(bytes.TrimPrefix(data, []byte("\ufeff")))
	if !strings.HasPrefix(text, frontDelim+"\n") && !strings.HasPrefix(text, frontDelim+"\r\n") {
		return map[string]any{}, text
	}
	rest := text[strings.Index(text, "\n")+1:]
	end := -1
	for i, line := range strings.SplitAfter(rest, "\n") {
		if strings.TrimRight(line, "\r\n") == frontDelim {
			end = i
			break
		}
	}
	if end < 0 {
		return map[string]any{}, text
	}
	lines := strings.SplitAfter(rest, "\n")
	head := strings.Join(lines[:end], "")
	body := strings.Join(lines[end+1:], "")
	meta := map[string]any{}
	if err := yaml.Unmarshal([]byte(head), &meta); err != nil || meta == nil {
		return map[string]any{}, body
	}
	return meta, body
}

// MetaString is a front matter value as text: a date without a time as
// 2006-01-02, any other non-string as its YAML form.
func MetaString(meta map[string]any, key string) string {
	switch v := meta[key].(type) {
	case string:
		return strings.TrimSpace(v)
	case nil:
		return ""
	case time.Time:
		if v.Hour() == 0 && v.Minute() == 0 && v.Second() == 0 {
			return v.Format(time.DateOnly)
		}
		return v.Format(time.RFC3339)
	default:
		b, _ := yaml.Marshal(v)
		return strings.TrimSpace(string(b))
	}
}

// MetaList is a front matter list: a YAML list, or one comma-separated string.
// A leading "#" is dropped from each entry, and empty entries are left out.
func MetaList(meta map[string]any, key string) []string {
	switch v := meta[key].(type) {
	case string:
		var out []string
		for _, s := range strings.Split(v, ",") {
			if s = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(s), "#")); s != "" {
				out = append(out, s)
			}
		}
		return out
	case []any:
		out := make([]string, 0, len(v))
		for _, x := range v {
			if s, ok := x.(string); ok {
				if s = strings.TrimSpace(strings.TrimPrefix(s, "#")); s != "" {
					out = append(out, s)
				}
			}
		}
		return out
	}
	return nil
}
