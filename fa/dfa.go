package fa

import (
	"fmt"
	"iter"
	"maps"
	"slices"
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
	slices.Sort(set)
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

	for c, part := range b.d.part.All() {
		if to := b.n.step(set, part.Min()); len(to) > 0 {
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

	return slices.Sorted(maps.Keys(ends)), slices.Sorted(maps.Keys(goes))
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

func (d *DFA[S]) Partition() *Partition[S] { return d.part }

func (d *DFA[S]) Index() Index[S] { return d.index }

func (d *DFA[S]) States() int { return len(d.nodes) - 1 }

func (d *DFA[S]) All() iter.Seq[State] {
	return func(yield func(State) bool) {
		for s := State(1); int(s) < len(d.nodes); s++ {
			if !yield(s) {
				return
			}
		}
	}
}

func (d *DFA[S]) Start() State { return d.start }

func (d *DFA[S]) Next(state State, class Class) State { return d.nodes[state].next[class] }

func (d *DFA[S]) Accept(state State) Match {
	if done := d.nodes[state].done; len(done) > 0 {
		return done[0]
	}

	return NoMatch
}

func (d *DFA[S]) Done(state State) []Match { return slices.Clone(d.nodes[state].done) }

func (d *DFA[S]) Going(state State) []Match { return slices.Clone(d.nodes[state].going) }

func (d *DFA[S]) Edges(s State) iter.Seq2[State, Set[S]] {
	by := map[State]Set[S]{}

	for c, part := range d.part.All() {
		if to := d.nodes[s].next[c]; to != NoState {
			by[to] = by[to].Union(part)
		}
	}

	return func(yield func(State, Set[S]) bool) {
		for _, to := range slices.Sorted(maps.Keys(by)) {
			if !yield(to, by[to]) {
				return
			}
		}
	}
}

func (d *DFA[S]) Step(state State, x S) State { return d.nodes[state].next[d.index.Of(x)] }

func (d *DFA[S]) Accepted() []Match {
	in := map[Match]bool{}

	for s := range d.nodes {
		if a := d.Accept(State(s)); a != NoMatch {
			in[a] = true
		}
	}

	return slices.Sorted(maps.Keys(in))
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
