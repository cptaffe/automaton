package fa

import "slices"

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
	sets := make([]Set[S], 0, a.part.Len()+b.part.Len())

	for _, d := range []*DFA[S]{a, b} {
		for _, part := range d.part.All() {
			sets = append(sets, part)
		}
	}

	joint := Split(sets)
	out := make([]S, 0, joint.Len())

	for _, part := range joint.All() {
		out = append(out, part.Min())
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

	slices.Reverse(out)

	return out
}
