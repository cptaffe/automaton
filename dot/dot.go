package dot

import (
	"fmt"
	"sort"
	"strings"

	"github.com/cptaffe/automaton"
)

type Machine interface {
	Name() string
	Note() string
	States() int
	Starts() map[string]automaton.State
	Classes() *automaton.Classes
	Next(state automaton.State, class automaton.Class) automaton.State
	Accepts(state automaton.State) (label string, ok bool)
}

type Style struct {
	Dashed bool
	Class  string
}

type Edges interface {
	Style(state automaton.State, class automaton.Class) Style
}

type Named interface {
	Named(s automaton.Set) (string, bool)
}

type Leaves interface {
	Leaves(state automaton.State) (scope string, ok bool)
}

func label(m Machine, set automaton.Set) string {
	if n, ok := m.(Named); ok {
		if got, ok := n.Named(set); ok {
			return got
		}
	}

	return set.Label()
}

func Draw(m Machine) []byte {
	g := Graph{
		Name: m.Name(),
		Note: m.Note(),
		Look: "machine",
	}

	starts := map[automaton.State]string{}

	for name, at := range m.Starts() {
		starts[at] = name
	}

	for s := automaton.State(0); int(s) < m.States(); s++ {
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

	style, _ := m.(Edges)

	for s := automaton.State(0); int(s) < m.States(); s++ {
		to := map[automaton.State][]automaton.Class{}

		for c := automaton.Class(0); int(c) < m.Classes().Len(); c++ {
			if at := m.Next(s, c); at != automaton.NoState {
				to[at] = append(to[at], c)
			}
		}

		for _, at := range sorted(to) {
			var set automaton.Set

			for _, c := range to[at] {
				set = set.Union(m.Classes().Part(c))
			}

			e := Edge{
				From:  fmt.Sprintf("s%d", s),
				To:    fmt.Sprintf("s%d", at),
				Label: " " + label(m, set) + " ",
			}

			if style != nil {
				st := style.Style(s, to[at][0])
				e.Dashed, e.Class = st.Dashed, st.Class
			}

			g.Edges = append(g.Edges, e)
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

func sorted(to map[automaton.State][]automaton.Class) []automaton.State {
	out := make([]automaton.State, 0, len(to))

	for at := range to {
		out = append(out, at)
	}

	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })

	return out
}
