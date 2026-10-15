package draw

import (
	"fmt"
	"strings"

	"github.com/cptaffe/automaton/dot"
	"github.com/cptaffe/automaton/fa"
	"github.com/cptaffe/automaton/lex/gen"
)

func Whole(f *gen.Flat, ms []gen.Machine) dot.Graph {
	all := make([]dot.Machine, len(ms))
	for i, m := range ms {
		all[i] = machine{m: m, names: f.Defs}
	}
	return dot.Whole(all, f.Name())
}

func Shape(f *gen.Flat) dot.Graph {
	rs := make([]dot.Rule, 0, len(f.Scopes))
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
	return dot.Shape(rs, f.Name())
}

func Summary(m gen.Machine) string {
	return fmt.Sprintf("%s, %s, %s, %s", m.Scope,
		dot.Plural(len(m.Rules), "rule"), dot.Plural(m.DFA.States(), "state"),
		dot.Plural(m.DFA.Partition().Len(), "class"))
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
	var out []dot.Move
	for to, set := range d.m.DFA.Edges(fa.State(s)) {
		out = append(out, dot.Move{To: int(to), Label: d.label(set)})
	}
	return out
}

func (d machine) rule(s int) (gen.FlatRule, bool) {
	a := d.m.DFA.Accept(fa.State(s))
	if a == fa.NoMatch {
		return gen.FlatRule{}, false
	}
	return d.m.Rules[a-1], true
}

func (d machine) Accepts(s int) (string, bool) {
	r, ok := d.rule(s)
	if !ok {
		return "", false
	}
	var parts []string
	switch {
	case r.Acts.Emit != "":
		parts = append(parts, r.Acts.Emit)
	case r.Acts.Drop:
		parts = append(parts, "ignore")
	}
	switch {
	case r.Acts.Pop:
		parts = append(parts, "return")
	case r.Acts.Push != "":
		parts = append(parts, "(push "+r.Acts.Push+")")
	}
	return strings.Join(parts, " "), true
}

func (d machine) Leaves(s int) (string, bool) {
	r, ok := d.rule(s)
	return r.Acts.Goto, ok && r.Acts.Goto != "" && r.Acts.Goto != d.m.Scope
}

func (d machine) label(s fa.Set[rune]) string {
	if got, ok := d.named(s); ok {
		return got
	}
	return label(s)
}

func (d machine) named(s fa.Set[rune]) (string, bool) {
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
		say := def.Name + " \\ " + d.name(got.Without(s))
		if best == "" || len(say) < len(best) {
			best = say
		}
	}
	if best == "" || len(best) > wide || len(best) >= len(spelled(s)) {
		return "", false
	}
	return best, true
}

func (d machine) name(s fa.Set[rune]) string {
	for _, def := range d.names {
		if got, ok := def.Expr.Single(); ok && got.Equal(s) {
			return def.Name
		}
	}
	return "[" + spelled(s) + "]"
}
