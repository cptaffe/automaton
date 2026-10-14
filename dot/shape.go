package dot

import "regexp"

type Rule struct {
	Name  string
	Alts  []string
	Start bool
}

var word = regexp.MustCompile(`[A-Za-z_][A-Za-z_0-9.-]*`)

func Shape(rs []Rule, name string) Graph {
	gr := read(rs)
	loop := find(gr)

	g := Graph{
		Name: name,
		Look: "shape",
	}

	for _, r := range rs {
		g.Nodes = append(g.Nodes, r.node())
	}

	for _, r := range rs {
		for _, w := range gr.refs[r.Name] {
			if e, ok := loop.edge(r.Name, w); ok {
				g.Edges = append(g.Edges, e)
			}
		}
	}

	return g
}

func (r Rule) node() Node {
	label := r.Name

	for _, alt := range r.Alts {
		label += "\n  " + alt
	}

	n := Node{ID: quote(r.Name, "n"), Label: label, Style: "rounded", Flush: true}

	if r.Start {
		n.Class = "start"
	}

	return n
}

func (l loops) edge(from, to string) (Edge, bool) {
	at := l.group[from]
	e := Edge{From: quote(from, "n"), To: quote(to, "n")}

	switch {
	case at != 0 && l.group[to] == at:
		e.Class = "cycle"

		if l.embeds[at] {
			e.Class, e.Dashed = "cycle nest", true
		}
	case from == to:
		return e, false
	}

	return e, true
}
