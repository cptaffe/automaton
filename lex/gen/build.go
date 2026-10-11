package gen

import (
	"fmt"
	"sort"
	"strings"

	"github.com/cptaffe/automaton"
)

type Machine struct {
	Scope string
	DFA   *automaton.DFA
	Rules []FlatRule

	Covered []string
}

func Build(f *Flat) ([]Machine, []string, error) {
	var out []Machine
	live := map[string]bool{}

	var all []automaton.Rule

	for _, s := range f.Scopes {
		for _, r := range f.Of(s.Name) {
			all = append(all, automaton.Rule{Expr: r.Expr})
		}
	}

	classes := automaton.Partition(automaton.Sets(all))

	for _, s := range f.Scopes {
		rules := f.Of(s.Name)

		if len(rules) == 0 {
			return nil, nil, fmt.Errorf("%s:%d: %s has no rules", f.Name, s.Line, s.Name)
		}

		in := make([]automaton.Rule, len(rules))

		for i, r := range rules {
			in[i] = automaton.Rule{Expr: r.Expr, Name: label(r)}
		}

		d, err := automaton.BuildWith(in, classes)

		if err != nil {
			return nil, nil, fmt.Errorf("%s: in %s, %w", f.Name, s.Name, err)
		}

		d = d.Minimise()

		m := Machine{Scope: s.Name, DFA: d, Rules: rules}

		for i, r := range rules {
			if reaches(d, i) {
				live[key(r)] = true

				continue
			}

			if r.Scope != s.Name {
				continue
			}

			m.Covered = append(m.Covered, fmt.Sprintf("%s:%d: %s is covered in %s by %s",
				f.Name, r.Line, label(r), s.Name, covers(rules, d, i)))
		}

		out = append(out, m)
	}

	var dead []string

	for _, r := range f.Rules {
		if r.Include == "" && !live[key(r)] {
			dead = append(dead, fmt.Sprintf("%s:%d: %s is covered in every scope it runs in", f.Name, r.Line, label(r)))
		}
	}

	sort.Strings(dead)

	return out, dead, nil
}

func reaches(d *automaton.DFA, rule int) bool {
	for s := automaton.State(1); int(s) <= d.States(); s++ {
		if d.Accept(s) == automaton.Match(rule+1) {
			return true
		}
	}

	return false
}

func covers(rules []FlatRule, d *automaton.DFA, rule int) string {
	var out []string
	seen := map[string]bool{}

	for s := automaton.State(1); int(s) <= d.States(); s++ {
		a := d.Accept(s)

		if a == automaton.NoMatch || int(a) > rule {
			continue
		}

		if got := label(rules[a-1]); !seen[got] {
			seen[got] = true
			out = append(out, got)
		}
	}

	sort.Strings(out)

	if len(out) == 0 {
		return "nothing, so it reads text no rule can reach"
	}

	return strings.Join(out, " and ")
}

func key(r FlatRule) string { return fmt.Sprintf("%s:%d", r.Scope, r.Line) }

func label(r FlatRule) string {
	switch {
	case r.Acts.Emit != "":
		return r.Acts.Emit
	case r.Acts.Drop:
		return "ignore"
	case r.Acts.Pop:
		return "-> return"
	case r.Acts.Goto != "":
		return "-> " + r.Acts.Goto
	}

	return r.Acts.Do
}
