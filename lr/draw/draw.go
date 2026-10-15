package draw

import (
	"fmt"
	"sort"
	"strings"

	"github.com/cptaffe/automaton/dot"
	"github.com/cptaffe/automaton/fa"
	"github.com/cptaffe/automaton/lr"
)

func rules(g *lr.Grammar) []dot.Rule {
	out := make([]dot.Rule, len(g.Rules))
	for i, r := range g.Rules {
		out[i] = dot.Rule{Name: r.Name, Alts: r.Alts, Start: i == 0}
	}
	return out
}

func Shape(g *lr.Grammar, name string) dot.Graph {
	return dot.Shape(rules(g), name)
}

func Tokens(g *lr.Grammar, name string) (dot.Machine, error) {
	t, err := g.Tokens()
	if t == nil || err != nil {
		return nil, err
	}
	m := tokens{t: t, name: name}
	head := fmt.Sprintf("%s over tokens -- %s, %s", name, dot.Plural(m.States(), "state"), dot.Plural(len(t.Words()), "token"))
	nested := dot.Nested(rules(g))
	switch len(nested) {
	case 0:
		m.note = head + "; nothing self-embeds, so this is exactly the grammar"
	default:
		sort.Strings(nested)
		verb := "self-embed"
		if len(nested) == 1 {
			verb = "self-embeds"
		}
		m.note = fmt.Sprintf("%s; %s %s, so this reads more than the grammar does", head, strings.Join(nested, ", "), verb)
	}
	return m, nil
}

type tokens struct {
	t    *lr.Tokens
	name string
	note string
}

func (m tokens) Name() string { return m.name }
func (m tokens) Note() string { return m.note }
func (m tokens) States() int  { return m.t.DFA().States() }

func (m tokens) Starts() []dot.Start {
	return []dot.Start{{Name: m.name, State: int(m.t.DFA().Start())}}
}

func (m tokens) Accepts(s int) (string, bool) {
	return "", m.t.DFA().Accept(fa.State(s)) != fa.NoMatch
}

func (m tokens) Moves(s int) []dot.Move {
	d := m.t.DFA()
	by := map[fa.State][]string{}
	var order []fa.State
	for c := fa.Class(1); int(c) <= d.Classes().Len(); c++ {
		to := d.Next(fa.State(s), c)
		if to == fa.NoState {
			continue
		}
		if _, ok := by[to]; !ok {
			order = append(order, to)
		}
		by[to] = append(by[to], m.t.In(c)...)
	}
	sort.Slice(order, func(i, j int) bool { return order[i] < order[j] })
	out := make([]dot.Move, len(order))
	for i, to := range order {
		out[i] = dot.Move{To: int(to), Label: words(by[to])}
	}
	return out
}

func words(ws []string) string {
	sort.Strings(ws)
	if len(ws) > 4 {
		return fmt.Sprintf("%s and %d more", strings.Join(ws[:4], " "), len(ws)-4)
	}
	return strings.Join(ws, " ")
}
