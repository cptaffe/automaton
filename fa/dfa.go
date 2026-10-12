package fa

import (
	"fmt"
	"sort"
	"strings"
)

type DFA[S Symbol] struct {
	classes *Classes[S]
	nodes   []node
	start   State
}

type node struct {
	next  []State
	done  []Match
	going []Match
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
	d := &DFA[S]{classes: classes, nodes: []node{{next: make([]State, classes.Len()+1)}}}
	seen := map[string]int{}

	var add func(of []int) State

	add = func(of []int) State {
		set := n.closure(of)
		sort.Ints(set)
		key := fmt.Sprint(set)

		if at, ok := seen[key]; ok {
			return State(at)
		}

		at := State(len(d.nodes))
		seen[key] = int(at)
		d.nodes = append(d.nodes, node{})

		row := make([]State, d.classes.Len()+1)

		done, going := map[Match]bool{}, map[Match]bool{}

		for _, s := range set {
			if a := n.accept[s]; a != NoMatch {
				done[a] = true
			}

			for _, e := range n.out[s] {
				if !e.eps && n.owner[s] != NoMatch {
					going[n.owner[s]] = true
				}
			}
		}

		d.nodes[at].done, d.nodes[at].going = sorted(done), sorted(going)

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

		d.nodes[at].next = row

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

func (d *DFA[S]) States() int { return len(d.nodes) - 1 }

func (d *DFA[S]) Start() State { return d.start }

func (d *DFA[S]) Next(state State, class Class) State { return d.nodes[state].next[class] }

func (d *DFA[S]) Accept(state State) Match {
	if done := d.nodes[state].done; len(done) > 0 {
		return done[0]
	}

	return NoMatch
}

func (d *DFA[S]) Done(state State) []Match { return append([]Match(nil), d.nodes[state].done...) }

func (d *DFA[S]) Going(state State) []Match { return append([]Match(nil), d.nodes[state].going...) }

func (d *DFA[S]) Step(state State, x S) State { return d.nodes[state].next[d.classes.Of(x)] }

func (d *DFA[S]) Unreachable(rules []Rule[S]) []string {
	live := map[Match]bool{}

	for s := range d.nodes {
		live[d.Accept(State(s))] = true
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

	for s := 1; s < len(d.nodes); s++ {
		fmt.Fprintf(&b, "%d", s)

		if a := d.Accept(State(s)); a != NoMatch {
			fmt.Fprintf(&b, " accept %d", a)
		}

		for c, to := range d.nodes[s].next {
			if c > 0 && to != NoState {
				fmt.Fprintf(&b, " %s->%d", d.classes.Part(Class(c)), to)
			}
		}

		b.WriteString("\n")
	}

	return b.String()
}
