package draw

import (
	"fmt"

	"github.com/cptaffe/automaton"
	"github.com/cptaffe/automaton/dot"
	"github.com/cptaffe/automaton/lex/gen"
)

func Whole(f *gen.Flat, ms []gen.Machine) []byte {
	all := make([]dot.Machine, len(ms))
	for i, m := range ms {
		all[i] = machine{m: m, names: f.Defs}
	}
	return dot.Whole(all, f.Name)
}

func Shape(f *gen.Flat) []byte {
	var rs []dot.Rule
	for i, s := range f.Scopes {
		r := dot.Rule{Name: s.Name, Start: i == 0}
		if s.Includes != "" {
			r.Alts = append(r.Alts, "includes "+s.Includes)
		}
		for _, rule := range f.Rules {
			if rule.Scope == s.Name {
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

func (d machine) Starts() map[string]automaton.State {
	return map[string]automaton.State{d.m.Scope: d.m.DFA.Start()}
}

func (d machine) Classes() *automaton.Classes { return d.m.DFA.Classes() }

func (d machine) Next(s automaton.State, c automaton.Class) automaton.State {
	return d.m.DFA.Next(s, c)
}

func (d machine) Accepts(s automaton.State) (string, bool) {
	a := d.m.DFA.Accept(s)
	if a == automaton.NoMatch {
		return "", false
	}
	r := d.m.Rules[a]
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

func (d machine) Leaves(s automaton.State) (string, bool) {
	a := d.m.DFA.Accept(s)
	if a == automaton.NoMatch {
		return "", false
	}
	to := d.m.Rules[a].Acts.Goto
	return to, to != "" && to != d.m.Scope
}

func (d machine) Named(s automaton.Set) (string, bool) {
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
	if best == "" || len(best) > 24 || len(best) >= len(s.Spelled()) {
		return "", false
	}
	return best, true
}

func (d machine) name(s automaton.Set) string {
	for _, def := range d.names {
		if got, ok := def.Expr.Single(); ok && got.Equal(s) {
			return def.Name
		}
	}
	return "[" + s.Spelled() + "]"
}
