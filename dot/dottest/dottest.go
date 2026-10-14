package dottest

import (
	"strings"

	"github.com/cptaffe/automaton/dot"
)

func Nodes(g dot.Graph) []string {
	out := make([]string, 0, len(g.Nodes))

	for _, n := range nodes(g) {
		out = append(out, name(n)+kind(n))
	}

	return out
}

func Edges(g dot.Graph) []string {
	names := map[string]string{}

	for _, n := range nodes(g) {
		names[n.ID] = name(n)
	}

	out := make([]string, 0, len(g.Edges))

	for _, e := range g.Edges {
		s := names[e.From] + " -" + strings.TrimSpace(e.Label) + "-> " + names[e.To]

		if e.Class != "" {
			s += " (" + e.Class + ")"
		}

		out = append(out, s)
	}

	return out
}

func Groups(g dot.Graph) []string {
	out := make([]string, 0, len(g.Groups))

	for _, gr := range g.Groups {
		members := make([]string, 0, len(gr.Nodes))

		for _, n := range gr.Nodes {
			members = append(members, name(n)+kind(n))
		}

		out = append(out, strings.TrimSpace(gr.Label)+": "+strings.Join(members, ", "))
	}

	return out
}

func nodes(g dot.Graph) []dot.Node {
	out := append([]dot.Node(nil), g.Nodes...)

	for _, gr := range g.Groups {
		out = append(out, gr.Nodes...)
	}

	return out
}

func name(n dot.Node) string {
	return strings.Join(strings.Fields(strings.TrimSpace(n.Caption+"\n"+n.Label)), " ")
}

func kind(n dot.Node) string {
	switch {
	case n.Class != "" && n.Twice:
		return " (" + n.Class + ", accept)"
	case n.Class != "":
		return " (" + n.Class + ")"
	case n.Twice:
		return " (accept)"
	}

	return ""
}
