package fa

func Equal[S Symbol](a, b *DFA[S]) bool {
	_, differ := Distinguish(a, b)

	return !differ
}

func Distinguish[S Symbol](a, b *DFA[S]) ([]S, bool) {
	syms := samples(a, b)
	off := a.States() + 1
	parent := make([]int, off+b.States()+1)

	for i := range parent {
		parent[i] = i
	}

	find := func(x int) int {
		for parent[x] != x {
			parent[x] = parent[parent[x]]
			x = parent[x]
		}

		return x
	}

	work := []pair[S]{{p: a.Start(), q: b.Start(), from: -1}}
	parent[find(int(a.Start()))] = find(off + int(b.Start()))

	for i := 0; i < len(work); i++ {
		at := work[i]

		if a.Accept(at.p) != b.Accept(at.q) {
			return witness(work, i), true
		}

		for _, x := range syms {
			p, q := a.Step(at.p, x), b.Step(at.q, x)

			if rp, rq := find(int(p)), find(off+int(q)); rp != rq {
				parent[rp] = rq
				work = append(work, pair[S]{p: p, q: q, from: i, sym: x})
			}
		}
	}

	return nil, false
}

func samples[S Symbol](a, b *DFA[S]) []S {
	sets := make([]Set[S], 0, a.classes.Len()+b.classes.Len())

	for _, d := range []*DFA[S]{a, b} {
		for c := Class(1); int(c) <= d.classes.Len(); c++ {
			sets = append(sets, d.classes.Part(c))
		}
	}

	joint := Partition(sets)
	out := make([]S, 0, joint.Len())

	for c := Class(1); int(c) <= joint.Len(); c++ {
		out = append(out, joint.Part(c).r[0].Lo)
	}

	return out
}

type pair[S Symbol] struct {
	p, q State
	from int
	sym  S
}

func witness[S Symbol](work []pair[S], at int) []S {
	var out []S

	for ; work[at].from >= 0; at = work[at].from {
		out = append(out, work[at].sym)
	}

	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}

	return out
}
