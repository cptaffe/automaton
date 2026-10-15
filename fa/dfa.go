package fa

import (
	"fmt"
	"sort"
	"strings"
)

type DFA[S Symbol] struct {
	part  *Partition[S]
	index Index[S]
	nodes []node
	start State
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

const DefaultMaxStates = 10_000

func Build[S Symbol](rules []Rule[S]) (*DFA[S], error) {
	return BuildWith(rules, Split(Sets(rules)), 0)
}

func BuildWith[S Symbol](rules []Rule[S], part *Partition[S], max int) (*DFA[S], error) {
	if err := part.refine(Sets(rules)); err != nil {
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

		n.accept[end] = Match(i + 1)
	}

	return n.determinise(start, part, max)
}

func (n *nfa[S]) determinise(start int, part *Partition[S], max int) (*DFA[S], error) {
	if max <= 0 {
		max = DefaultMaxStates
	}

	d := &DFA[S]{part: part, index: part.Index(), nodes: []node{{next: make([]State, part.Len()+1)}}}
	b := &subsets[S]{n: n, d: d, seen: map[string]State{}, max: max}
	d.start = b.add([]int{start})

	if b.over {
		return nil, fmt.Errorf("the patterns need more than %d states to match together", max)
	}

	return d, nil
}

type subsets[S Symbol] struct {
	n    *nfa[S]
	d    *DFA[S]
	seen map[string]State
	max  int
	over bool
}

func (b *subsets[S]) add(of []int) State {
	set := b.n.closure(of)
	sort.Ints(set)
	key := fmt.Sprint(set)

	if at, ok := b.seen[key]; ok {
		return at
	}

	if b.over || len(b.d.nodes) > b.max {
		b.over = true

		return NoState
	}

	at := State(len(b.d.nodes))
	b.seen[key] = at
	b.d.nodes = append(b.d.nodes, node{})
	b.d.nodes[at].done, b.d.nodes[at].going = b.n.matches(set)

	row := make([]State, b.d.part.Len()+1)

	for c := Class(1); int(c) <= b.d.part.Len(); c++ {
		if to := b.n.step(set, b.d.part.Part(c).r[0].Lo); len(to) > 0 {
			row[c] = b.add(to)
		}
	}

	b.d.nodes[at].next = row

	return at
}

func (n *nfa[S]) matches(set []int) (done, going []Match) {
	ends, goes := map[Match]bool{}, map[Match]bool{}

	for _, s := range set {
		if a := n.accept[s]; a != NoMatch {
			ends[a] = true
		}

		for _, e := range n.out[s] {
			if !e.eps && n.owner[s] != NoMatch {
				goes[n.owner[s]] = true
			}
		}
	}

	return sorted(ends), sorted(goes)
}

func (n *nfa[S]) step(set []int, x S) []int {
	var to, weak []int

	for _, s := range set {
		for _, e := range n.out[s] {
			switch {
			case e.eps || !e.set.Has(x):
			case e.weak:
				weak = append(weak, e.to)
			default:
				to = append(to, e.to)
			}
		}
	}

	if len(to) == 0 {
		return weak
	}

	return to
}

func sorted(in map[Match]bool) []Match {
	out := make([]Match, 0, len(in))

	for m := range in {
		out = append(out, m)
	}

	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })

	return out
}

func (d *DFA[S]) Partition() *Partition[S] { return d.part }

func (d *DFA[S]) Index() Index[S] { return d.index }

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

func (d *DFA[S]) Step(state State, x S) State { return d.nodes[state].next[d.index.Of(x)] }

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
				fmt.Fprintf(&b, " %s->%d", d.part.Part(Class(c)), to)
			}
		}

		b.WriteString("\n")
	}

	return b.String()
}
