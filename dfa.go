package automaton

import (
	"fmt"
	"sort"
	"strings"
)

type DFA struct {
	classes *Classes
	next    [][]State
	accept  []Match
	start   State
}

type Rule struct {
	Expr Expr
	Name string
}

func Sets(rules []Rule) []Set {
	var out []Set

	for _, r := range rules {
		out = r.Expr.sets(out)
	}

	return out
}

func Build(rules []Rule) (*DFA, error) {
	return BuildWith(rules, Partition(Sets(rules)))
}

func BuildWith(rules []Rule, classes *Classes) (*DFA, error) {
	if err := classes.refine(Sets(rules)); err != nil {
		return nil, err
	}

	n := &nfa{}
	start := n.state()

	for i, r := range rules {
		in := n.state()
		n.eps(start, in)
		end := n.build(r.Expr, in)

		if end == in {
			return nil, fmt.Errorf("%s matches nothing at all, so it would never advance", r.Name)
		}

		if n.accept[end] == NoMatch {
			n.accept[end] = Match(i + 1)
		} else {
			done := n.state()
			n.eps(end, done)
			n.accept[done] = Match(i + 1)
		}
	}

	return n.determinise(start, classes), nil
}

func (n *nfa) determinise(start int, classes *Classes) *DFA {
	d := &DFA{classes: classes, next: [][]State{make([]State, classes.Len()+1)}, accept: []Match{NoMatch}}
	seen := map[string]int{}

	var add func(of []int) State

	add = func(of []int) State {
		set := n.closure(of)
		sort.Ints(set)
		key := fmt.Sprint(set)

		if at, ok := seen[key]; ok {
			return State(at)
		}

		at := State(len(d.next))
		seen[key] = int(at)
		d.next = append(d.next, nil)
		d.accept = append(d.accept, NoMatch)

		row := make([]State, d.classes.Len()+1)

		for _, s := range set {
			if a := n.accept[s]; a != NoMatch && (d.accept[at] == NoMatch || a < d.accept[at]) {
				d.accept[at] = a
			}
		}

		for c := Class(1); int(c) <= d.classes.Len(); c++ {
			var to []int
			lo := d.classes.Part(c).r[0].Lo

			for _, s := range set {
				for _, e := range n.out[s] {
					if !e.eps && e.set.Has(lo) {
						to = append(to, e.to)
					}
				}
			}

			if len(to) == 0 {
				continue
			}

			row[c] = add(to)
		}

		d.next[at] = row

		return at
	}

	d.start = add([]int{start})

	return d
}

func (d *DFA) Classes() *Classes { return d.classes }

func (d *DFA) States() int { return len(d.next) - 1 }

func (d *DFA) Start() State { return d.start }

func (d *DFA) Next(state State, class Class) State { return d.next[state][class] }

func (d *DFA) Accept(state State) Match { return d.accept[state] }

func (d *DFA) Step(state State, r rune) State { return d.next[state][d.classes.Of(r)] }

func (d *DFA) Unreachable(rules []Rule) []string {
	live := map[Match]bool{}

	for _, a := range d.accept {
		if a != NoMatch {
			live[a] = true
		}
	}

	var out []string

	for i, r := range rules {
		if !live[Match(i+1)] {
			out = append(out, r.Name)
		}
	}

	sort.Strings(out)

	return out
}

func (d *DFA) String() string {
	var b strings.Builder

	for s := 1; s < len(d.next); s++ {
		fmt.Fprintf(&b, "%d", s)

		if a := d.accept[s]; a != NoMatch {
			fmt.Fprintf(&b, " accept %d", a)
		}

		for c, to := range d.next[s] {
			if c > 0 && to != NoState {
				fmt.Fprintf(&b, " %s->%d", d.classes.Part(Class(c)), to)
			}
		}

		b.WriteString("\n")
	}

	return b.String()
}
