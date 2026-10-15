package lr

import (
	"testing"

	"github.com/cptaffe/automaton/fa"
)

func stackless(t *testing.T, ss []state, of map[string]int) *fa.DFA[int] {
	var n fa.NFA[int]

	at := map[int]int{}

	for _, s := range ss {
		at[s.at] = n.State()
	}

	lands := map[string][]int{}

	for _, s := range ss {
		for _, m := range s.moves {
			if m.jump {
				lands[m.on] = append(lands[m.on], m.to)
			}
		}
	}

	for _, s := range ss {
		for _, m := range s.moves {
			if m.jump {
				continue
			}

			r, ok := of[m.on]

			if !ok {
				continue
			}

			n.On(at[s.at], fa.One(r), at[m.to])
		}

		for _, name := range s.reduces {
			for _, to := range lands[name] {
				n.Eps(at[s.at], at[to])
			}
		}

		if s.accepts {
			n.Accepts(at[s.at], fa.Match(1))
		}
	}

	n.Start(at[ss[0].at])

	d, err := n.DFA()
	if err != nil {
		t.Fatal(err)
	}
	return d.Minimise()
}
