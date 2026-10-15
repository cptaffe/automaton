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

func Build(f *Flat, max int) ([]Machine, []string, error) {
	b := builder{flat: f, classes: f.alphabet(), live: map[int]bool{}, max: max}

	var out []Machine

	for _, s := range f.Scopes {
		m, err := b.machine(s)

		if err != nil {
			return nil, nil, err
		}

		out = append(out, m)
	}

	return out, b.dead(), nil
}

func (f *Flat) alphabet() *fa.Classes[rune] {
	var all []fa.Rule[rune]

	for _, s := range f.Scopes {
		for _, r := range f.Of(s.Name) {
			all = append(all, fa.Rule[rune]{Expr: r.Expr})
		}
	}

	return fa.Partition(fa.Sets(all))
}

type builder struct {
	flat    *Flat
	classes *fa.Classes[rune]
	live    map[int]bool
	max     int
}

func (b *builder) machine(s FlatScope) (Machine, error) {
	f := b.flat
	rules := f.Of(s.Name)

	if len(rules) == 0 {
		return Machine{}, fmt.Errorf("%s:%d: %s has no rules", f.Path, s.Line, s.Name)
	}

	in := make([]fa.Rule[rune], len(rules))

	for i, r := range rules {
		in[i] = fa.Rule[rune]{Expr: r.Expr, Name: label(r)}
	}

	d, err := fa.BuildWith(in, b.classes, b.max)

	if err != nil {
		return Machine{}, fmt.Errorf("%s: in %s, %w", f.Path, s.Name, err)
	}

	d = d.Minimise()

	m := Machine{Scope: s.Name, DFA: d, Rules: rules}

	for i, r := range rules {
		if reaches(d, i) {
			b.live[key(r)] = true

			continue
		}

		if r.Inlined {
			continue
		}

		m.Covered = append(m.Covered, fmt.Sprintf("%s:%d: %s is covered in %s by %s", f.Path, r.Line, label(r), s.Name, covers(rules, d, i)))
	}

	return m, nil
}

func (b *builder) dead() []string {
	var out []string

	told := map[int]bool{}

	for _, r := range b.flat.Rules {
		if r.Include == "" && !b.live[key(r)] && !told[key(r)] {
			told[key(r)] = true
			out = append(out, fmt.Sprintf("%s:%d: %s is covered in every scope it runs in", b.flat.Path, r.Line, label(r)))
		}
	}

	sort.Strings(out)

	return out
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
