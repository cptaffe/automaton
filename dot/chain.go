package dot

import (
	"fmt"
	"sort"

	"github.com/cptaffe/automaton"
)

type run struct {
	to  automaton.State
	set automaton.Set
	n   int
}

func through(m Machine) map[automaton.State]bool {
	starts := map[automaton.State]bool{}

	for _, s := range m.Starts() {
		starts[s] = true
	}

	out := map[automaton.State]bool{}

	for s := automaton.State(0); int(s) < m.States(); s++ {
		if starts[s] {
			continue
		}

		if _, ok := m.Accepts(s); ok {
			continue
		}

		if to, _, ok := only(m, s); ok && to != s {
			out[s] = true
		}
	}

	return out
}

func only(m Machine, s automaton.State) (automaton.State, automaton.Set, bool) {
	to, set, found := automaton.NoState, automaton.Set{}, false

	for c := automaton.Class(0); int(c) < m.Classes().Len(); c++ {
		got := m.Next(s, c)

		if got == automaton.NoState {
			continue
		}

		if found && got != to {
			return automaton.NoState, automaton.Set{}, false
		}

		to, found = got, true
		set = set.Union(m.Classes().Part(c))
	}

	return to, set, found
}

func follow(m Machine, skip map[automaton.State]bool, to automaton.State, set automaton.Set) run {
	got := run{to: to, set: set, n: 1}
	seen := map[automaton.State]bool{}

	for skip[got.to] && !seen[got.to] {
		seen[got.to] = true
		next, step, ok := only(m, got.to)

		if !ok || !step.Equal(got.set) {
			return got
		}

		got.to, got.n = next, got.n+1
	}

	return got
}

func kept(m Machine) map[automaton.State]bool {
	skip := through(m)

	for s := automaton.State(0); int(s) < m.States(); s++ {
		if skip[s] {
			continue
		}

		for _, r := range runs(m, skip, s) {
			delete(skip, r.to)
		}
	}

	return skip
}

func runs(m Machine, skip map[automaton.State]bool, s automaton.State) []run {
	to := map[automaton.State]automaton.Set{}
	var order []automaton.State

	for c := automaton.Class(0); int(c) < m.Classes().Len(); c++ {
		got := m.Next(s, c)

		if got == automaton.NoState {
			continue
		}

		if _, ok := to[got]; !ok {
			order = append(order, got)
		}

		to[got] = to[got].Union(m.Classes().Part(c))
	}

	sort.Slice(order, func(i, j int) bool { return order[i] < order[j] })
	out := make([]run, 0, len(order))

	for _, got := range order {
		out = append(out, follow(m, skip, got, to[got]))
	}

	return out
}

func times(label string, n int) string {
	if n == 1 {
		return label
	}

	return fmt.Sprintf("%s ×%d", label, n)
}
