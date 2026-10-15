package dot

import "fmt"

type run struct {
	move Move
	to   int
	n    int
}

func through(m Machine) map[int]bool {
	starts := map[int]bool{}

	for _, s := range m.Starts() {
		starts[s.State] = true
	}

	out := map[int]bool{}

	for s := 1; s <= m.States(); s++ {
		if starts[s] {
			continue
		}

		if _, ok := m.Accepts(s); ok {
			continue
		}

		if mv := m.Moves(s); len(mv) == 1 && mv[0].To != s {
			out[s] = true
		}
	}

	return out
}

func follow(m Machine, skip map[int]bool, mv Move) run {
	got := run{move: mv, to: mv.To, n: 1}
	seen := map[int]bool{}

	for skip[got.to] && !seen[got.to] {
		seen[got.to] = true
		next := m.Moves(got.to)

		if len(next) != 1 || !next[0].like(mv) {
			return got
		}

		got.to, got.n = next[0].To, got.n+1
	}

	return got
}

func folded(m Machine) map[int]bool {
	skip := through(m)

	for s := 1; s <= m.States(); s++ {
		if skip[s] {
			continue
		}

		for _, r := range runs(m, skip, s) {
			delete(skip, r.to)
		}
	}

	return skip
}

func runs(m Machine, skip map[int]bool, s int) []run {
	moves := m.Moves(s)
	out := make([]run, 0, len(moves))

	for _, mv := range moves {
		out = append(out, follow(m, skip, mv))
	}

	return out
}

func times(label string, n int) string {
	if n == 1 {
		return label
	}

	return fmt.Sprintf("%s ×%d", label, n)
}

func (mv Move) like(o Move) bool {
	return mv.Label == o.Label && mv.Class == o.Class && mv.Dashed == o.Dashed
}
