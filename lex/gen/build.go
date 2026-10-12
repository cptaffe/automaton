package gen

import (
	"fmt"
	"sort"
	"strings"

	"github.com/cptaffe/automaton/fa"
)

type Machine struct {
	Scope string
	DFA   *fa.DFA[rune]
	Rules []FlatRule

	Covered []string
}

func Build(f *Flat) ([]Machine, []string, error) {
	var out []Machine
	live := map[int]bool{}

	var all []fa.Rule[rune]

	for _, s := range f.Scopes {
		for _, r := range f.Of(s.Name) {
			all = append(all, fa.Rule[rune]{Expr: r.Expr})
		}
	}

	classes := fa.Partition(fa.Sets(all))

	for _, s := range f.Scopes {
		rules := f.Of(s.Name)

		if len(rules) == 0 {
			return nil, nil, fmt.Errorf("%s:%d: %s has no rules", f.Path, s.Line, s.Name)
		}

		in := make([]fa.Rule[rune], len(rules))

		for i, r := range rules {
			in[i] = fa.Rule[rune]{Expr: r.Expr, Name: label(r)}
		}

		d, err := fa.BuildWith(in, classes)

		if err != nil {
			return nil, nil, fmt.Errorf("%s: in %s, %w", f.Path, s.Name, err)
		}

		d = d.Minimise()

		m := Machine{Scope: s.Name, DFA: d, Rules: rules}

		for i, r := range rules {
			if reaches(d, i) {
				live[key(r)] = true

				continue
			}

			if r.Inlined {
				continue
			}

			m.Covered = append(m.Covered, fmt.Sprintf("%s:%d: %s is covered in %s by %s", f.Path, r.Line, label(r), s.Name, covers(rules, d, i)))
		}

		out = append(out, m)
	}

	var dead []string

	told := map[int]bool{}

	for _, r := range f.Rules {
		if r.Include == "" && !live[key(r)] && !told[key(r)] {
			told[key(r)] = true
			dead = append(dead, fmt.Sprintf("%s:%d: %s is covered in every scope it runs in", f.Path, r.Line, label(r)))
		}
	}

	sort.Strings(dead)

	return out, dead, nil
}

func reaches(d *fa.DFA[rune], rule int) bool {
	for s := fa.State(1); int(s) <= d.States(); s++ {
		if d.Accept(s) == fa.Match(rule+1) {
			return true
		}
	}

	return false
}

func covers(rules []FlatRule, d *fa.DFA[rune], rule int) string {
	var out []string
	seen := map[string]bool{}

	for s := fa.State(1); int(s) <= d.States(); s++ {
		a := d.Accept(s)

		if a == fa.NoMatch || int(a) > rule {
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

func key(r FlatRule) int { return r.Line }

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
