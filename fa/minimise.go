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
	next := make([][]int, len(d.next))
	sig := make([]string, len(d.next))
	for s, row := range d.next {
		next[s] = make([]int, len(row))
		for c, to := range row {
			next[s][c] = int(to)
		}
		sig[s] = fmt.Sprint(d.Accept(State(s)))
	}
	to := Reachable(next, Equivalent(next, sig), int(d.start))

	n := 0
	for _, t := range to {
		n = max(n, t)
	}
	out := &DFA[S]{classes: d.classes, start: State(to[d.start]), next: make([][]State, n+1),
		done: make([][]Match, n+1), going: make([][]Match, n+1)}
	for s := range out.next {
		out.next[s] = make([]State, d.classes.Len()+1)
	}
	for s, row := range d.next {
		if s != 0 && to[s] == 0 {
			continue
		}
		for c, t := range row {
			out.next[to[s]][c] = State(to[t])
		}
		out.done[to[s]] = union(out.done[to[s]], d.done[s])
		out.going[to[s]] = union(out.going[to[s]], d.going[s])
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
