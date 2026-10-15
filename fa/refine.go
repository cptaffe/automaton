package fa

import (
	"cmp"
	"slices"
)

type refinable struct {
	sets    int
	elems   []int
	loc     []int
	set     []int
	first   []int
	past    []int
	marked  []int
	touched []int
}

func newRefinable(n int) *refinable {
	r := &refinable{elems: make([]int, n), loc: make([]int, n), set: make([]int, n),
		first: make([]int, n+1), past: make([]int, n+1), marked: make([]int, n+1)}
	for i := range n {
		r.elems[i], r.loc[i] = i, i
	}
	if n > 0 {
		r.sets, r.past[0] = 1, n
	}
	return r
}

func (r *refinable) mark(e int) {
	s, i := r.set[e], r.loc[e]
	j := r.first[s] + r.marked[s]
	r.elems[i], r.loc[r.elems[j]] = r.elems[j], i
	r.elems[j], r.loc[e] = e, j
	if r.marked[s] == 0 {
		r.touched = append(r.touched, s)
	}
	r.marked[s]++
}

func (r *refinable) split() {
	for len(r.touched) > 0 {
		s := r.touched[len(r.touched)-1]
		r.touched = r.touched[:len(r.touched)-1]
		j := r.first[s] + r.marked[s]
		r.marked[s] = 0
		if j == r.past[s] {
			continue
		}
		z := r.sets
		if j-r.first[s] <= r.past[s]-j {
			r.first[z], r.past[z], r.first[s] = r.first[s], j, j
		} else {
			r.first[z], r.past[z], r.past[s] = j, r.past[s], j
		}
		for i := r.first[z]; i < r.past[z]; i++ {
			r.set[r.elems[i]] = z
		}
		r.sets++
	}
}

type transition struct{ tail, label, head int }

func Equivalent(next [][]int, sig []string) []int {
	ts := transitions(next, alive(next, sig))
	states, cords := bySig(sig), byLabel(ts)
	in := make([][]int, len(next))
	for i, t := range ts {
		in[t.head] = append(in[t.head], i)
	}
	for b, c := 1, 0; c < cords.sets; c++ {
		for i := cords.first[c]; i < cords.past[c]; i++ {
			states.mark(ts[cords.elems[i]].tail)
		}
		states.split()
		for ; b < states.sets; b++ {
			for i := states.first[b]; i < states.past[b]; i++ {
				for _, t := range in[states.elems[i]] {
					cords.mark(t)
				}
			}
			cords.split()
		}
	}
	return states.numbers()
}

func transitions(next [][]int, live []bool) []transition {
	var ts []transition
	for s, row := range next {
		for k, t := range row {
			if live[s] && live[t] {
				ts = append(ts, transition{s, k, t})
			}
		}
	}
	return ts
}

func bySig(sig []string) *refinable {
	r := newRefinable(len(sig))
	groups := map[string][]int{}
	var order []string
	for s, g := range sig {
		if _, ok := groups[g]; !ok {
			order = append(order, g)
		}
		groups[g] = append(groups[g], s)
	}
	for _, g := range order[1:] {
		for _, s := range groups[g] {
			r.mark(s)
		}
		r.split()
	}
	return r
}

func byLabel(ts []transition) *refinable {
	r := newRefinable(len(ts))
	slices.SortStableFunc(r.elems, func(a, b int) int { return cmp.Compare(ts[a].label, ts[b].label) })
	for i, t := range r.elems {
		if i > 0 && ts[t].label != ts[r.elems[i-1]].label {
			r.past[r.sets-1], r.first[r.sets] = i, i
			r.sets++
		}
		r.set[t], r.loc[t] = r.sets-1, i
	}
	if len(ts) > 0 {
		r.past[r.sets-1] = len(ts)
	}
	return r
}

func (r *refinable) numbers() []int {
	out := make([]int, len(r.set))
	at := map[int]int{r.set[0]: 0}
	for e, s := range r.set {
		if _, ok := at[s]; !ok {
			at[s] = len(at)
		}
		out[e] = at[s]
	}
	return out
}

func alive(next [][]int, sig []string) []bool {
	back := make([][]int, len(next))
	for s, row := range next {
		for _, t := range row {
			back[t] = append(back[t], s)
		}
	}
	live := make([]bool, len(next))
	var work []int
	for s := range next {
		if sig[s] != sig[0] {
			live[s] = true
			work = append(work, s)
		}
	}
	for len(work) > 0 {
		s := work[len(work)-1]
		work = work[:len(work)-1]
		for _, p := range back[s] {
			if !live[p] {
				live[p] = true
				work = append(work, p)
			}
		}
	}
	return live
}
