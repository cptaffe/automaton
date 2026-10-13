package dot

import "regexp"

type Rule struct {
	Name  string
	Alts  []string
	Start bool
}

var word = regexp.MustCompile(`[A-Za-z_][A-Za-z_0-9.-]*`)

func Shape(rs []Rule, name string) ([]byte, error) {
	gr := read(rs)
	loop := find(gr)

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
				if !gr.is[w] || seen[w] {
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
