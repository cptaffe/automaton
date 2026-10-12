package draw

import (
	"fmt"
	"sort"

	"github.com/cptaffe/automaton"
	"github.com/cptaffe/automaton/dot"
	"github.com/cptaffe/automaton/lex/gen"
)

func Whole(f *gen.Flat, ms []gen.Machine) ([]byte, error) {
	all := make([]dot.Machine, len(ms))
	for i, m := range ms {
		all[i] = machine{m: m, names: f.Defs}
	}
	return dot.Whole(all, f.Name)
}

func Shape(f *gen.Flat) ([]byte, error) {
	var rs []dot.Rule
	for i, s := range f.Scopes {
		r := dot.Rule{Name: s.Name, Start: i == 0}
		for _, rule := range f.Rules {
			switch {
			case rule.Scope != s.Name:
			case rule.Include != "":
				r.Alts = append(r.Alts, "-> "+rule.Include)
			default:
				r.Alts = append(r.Alts, rule.Text)
			}
		}
		rs = append(rs, r)
	}
	return dot.Shape(rs, f.Name)
}

func Summary(m gen.Machine) string {
	return fmt.Sprintf("%s, %s, %s, %s", m.Scope,
		dot.Plural(len(m.Rules), "rule"), dot.Plural(m.DFA.States(), "state"),
		dot.Plural(m.DFA.Classes().Len(), "class"))
}

type machine struct {
	m     gen.Machine
	names []gen.Def
}

func (d machine) Name() string { return d.m.Scope }

func (d machine) Note() string {
	s := Summary(d.m)
	return d.m.Scope + " --" + s[len(d.m.Scope)+1:]
}

func (d machine) States() int { return d.m.DFA.States() }

func (d machine) Starts() []dot.Start {
	return []dot.Start{{Name: d.m.Scope, State: int(d.m.DFA.Start())}}
}

func (d machine) Moves(s int) []dot.Move {
	by := map[automaton.State]automaton.Set[rune]{}
	var order []automaton.State
	c := d.m.DFA.Classes()
	for k := automaton.Class(1); int(k) <= c.Len(); k++ {
		to := d.m.DFA.Next(automaton.State(s), k)
		if to == automaton.NoState {
			continue
		}
		if _, ok := by[to]; !ok {
			order = append(order, to)
		}
		by[to] = by[to].Union(c.Part(k))
	}
	sort.Slice(order, func(i, j int) bool { return order[i] < order[j] })
	out := make([]dot.Move, len(order))
	for i, to := range order {
		out[i] = dot.Move{To: int(to), Label: d.label(by[to])}
	}
	return out
}

func (d machine) rule(s int) (gen.FlatRule, bool) {
	a := d.m.DFA.Accept(automaton.State(s))
	if a == automaton.NoMatch {
		return gen.FlatRule{}, false
	}
	return d.m.Rules[a-1], true
}

func (d machine) Accepts(s int) (string, bool) {
	r, ok := d.rule(s)
	if !ok {
		return "", false
	}
	label := "-> " + r.Acts.Goto
	switch {
	case r.Acts.Emit != "":
		label = r.Acts.Emit
	case r.Acts.Drop:
		label = "ignore"
	}
	switch {
	case r.Acts.Pop:
		label += " -> return"
	case r.Acts.Push != "":
		label += " (push " + r.Acts.Push + ")"
	}
	return label, true
}

func (d machine) Leaves(s int) (string, bool) {
	r, ok := d.rule(s)
	return r.Acts.Goto, ok && r.Acts.Goto != "" && r.Acts.Goto != d.m.Scope
}

func (d machine) label(s automaton.Set[rune]) string {
	if got, ok := d.named(s); ok {
		return got
	}
	return label(s)
}

func (d machine) named(s automaton.Set[rune]) (string, bool) {
	best := ""
	for _, def := range d.names {
		got, ok := def.Expr.Single()
		if !ok || got.Empty() {
			continue
		}
		if got.Equal(s) {
			return def.Name, true
		}
		if !s.Without(got).Empty() {
			continue
		}
		say := def.Name + " - " + d.name(got.Without(s))
		if best == "" || len(say) < len(best) {
			best = say
		}
	}
	if best == "" || len(best) > wide || len(best) >= len(spelled(s)) {
		return "", false
	}
	return best, true
}

func (d machine) name(s automaton.Set[rune]) string {
	for _, def := range d.names {
		if got, ok := def.Expr.Single(); ok && got.Equal(s) {
			return def.Name
		}
	}
	return "[" + spelled(s) + "]"
}
