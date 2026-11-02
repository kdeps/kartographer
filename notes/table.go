package notes

import "strings"

// BodyParts is a note body split into the rows of its pipe tables and the
// count of its other lines: those neither blank nor a heading.
type BodyParts struct {
	Rows       [][]string // each table row's trimmed cells; separator and header rows are left out
	ProseLines int
}

// SplitBody splits body into table rows and prose lines. A row whose first
// cell equals header's first name, ignoring case, is the header and is left out.
func SplitBody(body string, header []string) BodyParts {
	var p BodyParts
	for _, line := range strings.Split(body, "\n") {
		t := strings.TrimSpace(line)
		switch {
		case t == "", strings.HasPrefix(t, "#"):
		case strings.HasPrefix(t, "|"):
			cells := strings.Split(strings.Trim(t, "|"), "|")
			for i := range cells {
				cells[i] = strings.TrimSpace(cells[i])
			}
			if strings.Trim(cells[0], "-: ") == "" {
				continue // separator row
			}
			if len(header) > 0 && strings.EqualFold(cells[0], header[0]) {
				continue
			}
			p.Rows = append(p.Rows, cells)
		default:
			p.ProseLines++
		}
	}
	return p
}

// AppendTableRow adds a "| cell | cell |" row at the end of text. withHeader
// writes header and its separator row first, for a note with no table yet.
func AppendTableRow(text string, header, cells []string, withHeader bool) string {
	text = strings.TrimRight(text, "\n")
	if withHeader {
		text += "\n| " + strings.Join(header, " | ") + " |\n|" + strings.Repeat("---|", len(header))
	}
	return text + "\n| " + strings.Join(cells, " | ") + " |\n"
}
