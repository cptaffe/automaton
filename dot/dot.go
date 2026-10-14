package dot

import (
	"fmt"
	"strings"
)

type Machine interface {
	Name() string
	Note() string
	States() int
	Starts() []Start
	Moves(state int) []Move
	Accepts(state int) (label string, ok bool)
}

type Start struct {
	Name  string
	State int
}

type Move struct {
	To     int
	Label  string
	Class  string
	Dashed bool
}

type Leaves interface {
	Leaves(state int) (scope string, ok bool)
}

func Draw(m Machine) Graph {
	g := Graph{
		Name: m.Name(),
		Note: m.Note(),
		Look: "machine",
	}

	starts := map[int]string{}

	for _, s := range m.Starts() {
		if _, ok := starts[s.State]; !ok {
			starts[s.State] = s.Name
		}
	}

	for s := 1; s <= m.States(); s++ {
		name := starts[s]
		g.Nodes = append(g.Nodes, state(fmt.Sprintf("s%d", s), s, name, name != "", m))
	}

	for s := 1; s <= m.States(); s++ {
		for _, mv := range m.Moves(s) {
			g.Edges = append(g.Edges, Edge{
				From:   fmt.Sprintf("s%d", s),
				To:     fmt.Sprintf("s%d", mv.To),
				Label:  " " + mv.Label + " ",
				Dashed: mv.Dashed,
				Class:  mv.Class,
			})
		}
	}

	return g
}

func state(id string, s int, name string, start bool, m Machine) Node {
	n := Node{ID: id, Label: fmt.Sprint(s)}

	if start {
		n.Label, n.Shape, n.Style, n.Class = name, "box", "rounded", "start"
	}

	got, ok := m.Accepts(s)

	if !ok {
		return n
	}

	n.Twice = true

	if got != "" {
		n.Caption, n.Label = n.Label, got
		n.Shape, n.Style = "box", "rounded"
	}

	return n
}

func quote(s, brk string) string {
	var b strings.Builder

	b.WriteByte('"')

	for _, r := range s {
		switch r {
		case '"', '\\':
			b.WriteByte('\\')
			b.WriteRune(r)
		case '\n':
			b.WriteByte('\\')
			b.WriteString(brk)
		default:
			b.WriteRune(r)
		}
	}

	b.WriteByte('"')

	return b.String()
}
