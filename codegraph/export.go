package codegraph

import (
	"bytes"
	"encoding/xml"
	"io"
	"sort"
	"strconv"
)

// Edges lists the call edges between indexed symbols as (caller, callee)
// pairs, sorted.
func (ix *Index) Edges() [][2]string {
	var out [][2]string
	for from, tos := range ix.deps.NodeDependencies {
		if _, ok := ix.Symbols[from]; !ok {
			continue
		}
		for _, to := range tos {
			if _, ok := ix.Symbols[to]; ok {
				out = append(out, [2]string{from, to})
			}
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i][0] != out[j][0] {
			return out[i][0] < out[j][0]
		}
		return out[i][1] < out[j][1]
	})
	return out
}

// WriteGraphML writes the symbols and call edges as a GraphML document, with
// each node's file, line and language as data.
func WriteGraphML(w io.Writer, ix *Index) error {
	if _, err := io.WriteString(w, `<?xml version="1.0" encoding="UTF-8"?>
<graphml xmlns="http://graphml.graphdrawing.org/xmlns">
<key id="file" for="node" attr.name="file" attr.type="string"/>
<key id="line" for="node" attr.name="line" attr.type="int"/>
<key id="language" for="node" attr.name="language" attr.type="string"/>
<key id="confidence" for="edge" attr.name="confidence" attr.type="string"/>
<graph id="G" edgedefault="directed">
`); err != nil {
		return err
	}
	names := make([]string, 0, len(ix.Symbols))
	for n := range ix.Symbols {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		s := ix.Symbols[n]
		if _, err := io.WriteString(w, `<node id="`+escape(n)+`">`+
			`<data key="file">`+escape(s.File)+`</data>`+
			`<data key="line">`+escape(itoa(s.Line))+`</data>`+
			`<data key="language">`+escape(s.Language)+`</data></node>`+"\n"); err != nil {
			return err
		}
	}
	for i, e := range ix.Edges() {
		if _, err := io.WriteString(w, `<edge id="e`+itoa(i)+`" source="`+escape(e[0])+`" target="`+escape(e[1])+`"><data key="confidence">`+ix.Confidence(e[0], e[1])+`</data></edge>`+"\n"); err != nil {
			return err
		}
	}
	_, err := io.WriteString(w, "</graph>\n</graphml>\n")
	return err
}

func escape(s string) string {
	var b bytes.Buffer
	_ = xml.EscapeText(&b, []byte(s))
	return b.String()
}

func itoa(n int) string {
	return strconv.Itoa(n)
}
