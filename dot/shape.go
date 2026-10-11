package dot

import (
	"regexp"
	"strings"
)

type Rule struct {
	Name  string
	Alts  []string
	Start bool
}

var word = regexp.MustCompile(`[A-Za-z_][A-Za-z_0-9.-]*`)

func Shape(rs []Rule, name string) []byte {
	is := map[string]bool{}

	for _, r := range rs {
		is[r.Name] = true
	}

	loop := find(rs, is)

	g := Graph{
		Name: name,
		Look: "shape",
	}

	for _, r := range rs {
		label := r.Name

		for _, alt := range r.Alts {
			label += "\n  " + alt
		}

		n := Node{ID: quote(r.Name, "n"), Label: label, Style: "rounded", Flush: true}

		if r.Start {
			n.Class = "start"
		}

		g.Nodes = append(g.Nodes, n)
	}

	for _, r := range rs {
		seen := map[string]bool{}

		for _, alt := range r.Alts {
			for _, w := range word.FindAllString(alt, -1) {
				if !is[w] || seen[w] {
					continue
				}

				seen[w] = true

				at := loop.group[r.Name]
				e := Edge{From: quote(r.Name, "n"), To: quote(w, "n")}

				switch {
				case at != 0 && loop.group[w] == at:
					e.Class = "cycle"

					if loop.embeds[at] {
						e.Class, e.Dashed = "cycle nest", true
					}
				case w == r.Name:
					continue
				}

				g.Edges = append(g.Edges, e)
			}
		}
	}

	return g.Bytes()
}

func lines(s string) string {
	var b strings.Builder

	b.WriteByte('"')

	for _, r := range s {
		switch r {
		case '"', '\\':
			b.WriteByte('\\')
			b.WriteRune(r)
		case '\n':
			b.WriteString("\\l")
		default:
			b.WriteRune(r)
		}
	}

	b.WriteString("\\l\"")

	return b.String()
}
