package fa

import (
	"fmt"
	"slices"
)

func Reachable(next [][]int, eq []int, starts ...int) []int {
	out := make([]int, len(next))
	at := map[int]int{0: 0}
	work := slices.Clone(starts)
	for len(work) > 0 {
		s := work[0]
		work = work[1:]
		if _, ok := at[eq[s]]; ok {
			continue
		}
		at[eq[s]] = len(at)
		work = append(work, next[s]...)
	}
	for s := range next {
		if n, ok := at[eq[s]]; ok {
			out[s] = n
		}
	}
	return out
}

func (d *DFA[S]) Minimise() *DFA[S] {
	next := make([][]int, len(d.nodes))
	sig := make([]string, len(d.nodes))
	for s, nd := range d.nodes {
		next[s] = make([]int, len(nd.next))
		for c, to := range nd.next {
			next[s][c] = int(to)
		}
		sig[s] = fmt.Sprint(d.Accept(State(s)))
	}
	to := Reachable(next, Equivalent(next, sig), int(d.start))

	n := slices.Max(to)
	out := &DFA[S]{part: d.part, index: d.index, start: State(to[d.start]), nodes: make([]node, n+1)}
	for s := range out.nodes {
		out.nodes[s].next = make([]State, d.part.Len()+1)
	}
	for s, nd := range d.nodes {
		if s != 0 && to[s] == 0 {
			continue
		}
		m := &out.nodes[to[s]]
		for c, t := range nd.next {
			m.next[c] = State(to[t])
		}
		m.done = union(m.done, nd.done)
		m.going = union(m.going, nd.going)
	}
	return out
}

func union(a, b []Match) []Match {
	out := slices.Concat(a, b)
	slices.Sort(out)

	return slices.Compact(out)
}
