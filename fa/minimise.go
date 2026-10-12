package fa

import (
	"fmt"
	"strconv"
	"strings"
)

func Equivalent(next [][]int, sig []string) []int {
	group := make([]int, len(next))
	by := map[string]int{}
	for s := range next {
		group[s] = number(by, sig[s])
	}
	for {
		by = map[string]int{}
		again := make([]int, len(next))
		var key strings.Builder
		for s, row := range next {
			key.Reset()
			key.WriteString(strconv.Itoa(group[s]))
			for _, to := range row {
				key.WriteByte(' ')
				key.WriteString(strconv.Itoa(group[to]))
			}
			again[s] = number(by, key.String())
		}
		if len(by) == count(group) {
			break
		}
		group = again
	}

	out := make([]int, len(next))
	at := map[int]int{group[0]: 0}
	for s := range next {
		if _, ok := at[group[s]]; !ok {
			at[group[s]] = len(at)
		}
		out[s] = at[group[s]]
	}
	return out
}

func number(by map[string]int, key string) int {
	n, ok := by[key]
	if !ok {
		n = len(by)
		by[key] = n
	}
	return n
}

func count(group []int) int {
	seen := map[int]bool{}
	for _, g := range group {
		seen[g] = true
	}
	return len(seen)
}

func Reachable(next [][]int, eq []int, starts ...int) []int {
	out := make([]int, len(next))
	at := map[int]int{0: 0}
	work := append([]int(nil), starts...)
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

	n := 0
	for _, t := range to {
		n = max(n, t)
	}
	out := &DFA[S]{classes: d.classes, start: State(to[d.start]), nodes: make([]node, n+1)}
	for s := range out.nodes {
		out.nodes[s].next = make([]State, d.classes.Len()+1)
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
	in := map[Match]bool{}

	for _, m := range a {
		in[m] = true
	}

	for _, m := range b {
		in[m] = true
	}

	return sorted(in)
}
