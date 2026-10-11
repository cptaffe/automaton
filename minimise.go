package automaton

import "sort"

func (d *DFA) Minimise() *DFA {
	sink := len(d.next)
	n := sink + 1
	cols := d.classes.Len()

	to := make([][]int, n)

	for s := range to {
		to[s] = make([]int, cols)

		for c := range to[s] {
			at := sink

			if s != sink && d.next[s][Class(c)] != NoState {
				at = int(d.next[s][Class(c)])
			}

			to[s][c] = at
		}
	}

	accept := func(s int) Match {
		if s == sink {
			return NoMatch
		}

		return d.accept[s]
	}

	group := make([]int, n)
	byAccept := map[Match]int{}

	for s := 0; s < n; s++ {
		a := accept(s)
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
				if in[to[s][c]] {
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

	return d.rebuild(group, parts, sink)
}

func (d *DFA) rebuild(group []int, parts [][]int, sink int) *DFA {
	keep := map[int]bool{}

	var walk func(int)

	walk = func(g int) {
		if keep[g] {
			return
		}

		keep[g] = true

		for _, s := range parts[g] {
			if s == sink {
				continue
			}

			for c := range d.next[s] {
				if at := d.next[s][Class(c)]; at != NoState {
					walk(group[at])
				}
			}
		}
	}

	walk(group[d.start])

	live := make([]int, 0, len(keep))

	for g := range keep {
		live = append(live, g)
	}

	sort.Ints(live)

	at := map[int]int{}

	for i, g := range live {
		at[g] = i
	}

	out := &DFA{classes: d.classes, start: State(at[group[d.start]])}

	for _, g := range live {
		row := make([]State, d.classes.Len())

		for c := range row {
			row[c] = NoState
		}

		a := NoMatch

		for _, s := range parts[g] {
			if s == sink {
				continue
			}

			if d.accept[s] != NoMatch && (a == NoMatch || d.accept[s] < a) {
				a = d.accept[s]
			}

			for c := range d.next[s] {
				if to := d.next[s][Class(c)]; to != NoState {
					row[c] = State(at[group[to]])
				}
			}
		}

		out.next = append(out.next, row)
		out.accept = append(out.accept, a)
	}

	return out
}
