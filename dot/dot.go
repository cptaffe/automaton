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

func Draw(m Machine) ([]byte, error) {
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
		n := Node{ID: fmt.Sprintf("s%d", s), Label: fmt.Sprint(s)}

		if name, ok := starts[s]; ok {
			n.Label, n.Shape, n.Style, n.Class = name, "box", "rounded", "start"
		}

		if got, ok := m.Accepts(s); ok {
			n.Shape = "doublecircle"

			if got != "" {
				n.Label += "\n" + got
			}
		}

		g.Nodes = append(g.Nodes, n)
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

	return g.Bytes()
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
