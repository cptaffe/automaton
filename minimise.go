package automaton

import "sort"

func (d *DFA) Minimise() *DFA {
	n := len(d.next)
	cols := d.classes.Len() + 1

	group := make([]int, n)
	byAccept := map[Match]int{}

	for s := 0; s < n; s++ {
		a := d.accept[s]
		at, ok := byAccept[a]

		if !ok {
			at = len(byAccept)
			byAccept[a] = at
		}

		group[s] = at
	}

	parts := make([][]int, len(byAccept))

	for s := 0; s < n; s++ {
		parts[group[s]] = append(parts[group[s]], s)
	}

	work := make([]int, len(parts))

	for i := range work {
		work[i] = i
	}

	for len(work) > 0 {
		a := work[len(work)-1]
		work = work[:len(work)-1]

		if a >= len(parts) || len(parts[a]) == 0 {
			continue
		}

		in := map[int]bool{}

		for _, s := range parts[a] {
			in[s] = true
		}

		for c := 0; c < cols; c++ {
			hit := map[int]bool{}

			for s := 0; s < n; s++ {
				if in[int(d.next[s][c])] {
					hit[s] = true
				}
			}

			if len(hit) == 0 {
				continue
			}

			for p := range parts {
				var yes, no []int

				for _, s := range parts[p] {
					if hit[s] {
						yes = append(yes, s)

						continue
					}

					no = append(no, s)
				}

				if len(yes) == 0 || len(no) == 0 {
					continue
				}

				parts[p] = yes
				parts = append(parts, no)
				at := len(parts) - 1

				for _, s := range no {
					group[s] = at
				}

				work = append(work, p, at)
			}
		}
	}

	return d.rebuild(group, parts)
}

func (d *DFA) rebuild(group []int, parts [][]int) *DFA {
	keep := map[int]bool{}

	var walk func(int)

	walk = func(g int) {
		if keep[g] {
			return
		}

		keep[g] = true

		for _, s := range parts[g] {
			for _, at := range d.next[s] {
				walk(group[at])
			}
		}
	}

	walk(group[d.start])

	dead := group[NoState]
	live := make([]int, 0, len(keep))

	for g := range keep {
		if g != dead {
			live = append(live, g)
		}
	}

	sort.Ints(live)

	at := map[int]State{dead: NoState}

	for i, g := range live {
		at[g] = State(i + 1)
	}

	out := &DFA{classes: d.classes, start: at[group[d.start]],
		next: [][]State{make([]State, d.classes.Len()+1)}, accept: []Match{NoMatch}}

	for _, g := range live {
		row := make([]State, d.classes.Len()+1)
		a := NoMatch

		for _, s := range parts[g] {
			if d.accept[s] != NoMatch && (a == NoMatch || d.accept[s] < a) {
				a = d.accept[s]
			}

			for c, to := range d.next[s] {
				if to != NoState {
					row[c] = at[group[to]]
				}
			}
		}

		out.next = append(out.next, row)
		out.accept = append(out.accept, a)
	}

	return out
}
