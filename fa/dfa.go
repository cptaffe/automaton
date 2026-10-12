package fa

import (
	"fmt"
	"sort"
	"strings"
)

type DFA[S Symbol] struct {
	classes *Classes[S]
	next    [][]State
	accept  []Match
	done    [][]Match
	going   [][]Match
	start   State
}

type Rule[S Symbol] struct {
	Expr Expr[S]
	Name string
}

func Sets[S Symbol](rules []Rule[S]) []Set[S] {
	var out []Set[S]

	for _, r := range rules {
		out = r.Expr.sets(out)
	}

	return out
}

func Build[S Symbol](rules []Rule[S]) (*DFA[S], error) {
	return BuildWith(rules, Partition(Sets(rules)))
}

func BuildWith[S Symbol](rules []Rule[S], classes *Classes[S]) (*DFA[S], error) {
	if err := classes.refine(Sets(rules)); err != nil {
		return nil, err
	}

	n := &nfa[S]{}
	start := n.state()

	for i, r := range rules {
		n.rule = Match(i + 1)
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

func (n *nfa[S]) determinise(start int, classes *Classes[S]) *DFA[S] {
	d := &DFA[S]{classes: classes, next: [][]State{make([]State, classes.Len()+1)}, accept: []Match{NoMatch}, done: [][]Match{nil}, going: [][]Match{nil}}
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
		d.done = append(d.done, nil)
		d.going = append(d.going, nil)

		row := make([]State, d.classes.Len()+1)

		done, going := map[Match]bool{}, map[Match]bool{}

		for _, s := range set {
			if a := n.accept[s]; a != NoMatch {
				done[a] = true

				if d.accept[at] == NoMatch || a < d.accept[at] {
					d.accept[at] = a
				}
			}

			for _, e := range n.out[s] {
				if !e.eps && n.owner[s] != NoMatch {
					going[n.owner[s]] = true
				}
			}
		}

		d.done[at], d.going[at] = sorted(done), sorted(going)

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

func sorted(in map[Match]bool) []Match {
	out := make([]Match, 0, len(in))

	for m := range in {
		out = append(out, m)
	}

	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })

	return out
}

func (d *DFA[S]) Classes() *Classes[S] { return d.classes }

func (d *DFA[S]) States() int { return len(d.next) - 1 }

func (d *DFA[S]) Start() State { return d.start }

func (d *DFA[S]) Next(state State, class Class) State { return d.next[state][class] }

func (d *DFA[S]) Accept(state State) Match { return d.accept[state] }

func (d *DFA[S]) Done(state State) []Match { return append([]Match(nil), d.done[state]...) }

func (d *DFA[S]) Going(state State) []Match { return append([]Match(nil), d.going[state]...) }

func (d *DFA[S]) Step(state State, x S) State { return d.next[state][d.classes.Of(x)] }

func (d *DFA[S]) Unreachable(rules []Rule[S]) []string {
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

func (d *DFA[S]) String() string {
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
