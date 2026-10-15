package gen

import (
	"fmt"
	"slices"
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
	b := builder{flat: f, part: f.alphabet(), live: map[int]bool{}, max: max}

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

func (f *Flat) alphabet() *fa.Partition[rune] {
	var all []fa.Rule[rune]

	for _, s := range f.Scopes {
		for _, r := range f.Of(s.Name) {
			all = append(all, fa.Rule[rune]{Expr: r.Expr})
		}
	}

	return fa.Split(fa.Sets(all))
}

type builder struct {
	flat *Flat
	part *fa.Partition[rune]
	live map[int]bool
	max  int
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

	d, err := fa.BuildWith(in, b.part, b.max)

	if err != nil {
		return Machine{}, fmt.Errorf("%s: in %s, %w", f.Path, s.Name, err)
	}

	d = d.Minimise()

	m := Machine{Scope: s.Name, DFA: d, Rules: rules}
	accepted := d.Accepted()

	for i, r := range rules {
		if slices.Contains(accepted, fa.Match(i+1)) {
			b.live[key(r)] = true

			continue
		}

		if r.Inlined {
			continue
		}

		m.Covered = append(m.Covered, fmt.Sprintf("%s:%d: %s is covered in %s by %s", f.Path, r.Line, label(r), s.Name, covers(rules, accepted, i)))
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

	slices.Sort(out)

	return out
}

func covers(rules []FlatRule, accepted []fa.Match, rule int) string {
	var out []string

	for _, a := range accepted {
		if int(a) > rule {
			break
		}

		out = append(out, label(rules[a-1]))
	}

	slices.Sort(out)
	out = slices.Compact(out)

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
