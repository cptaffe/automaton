package fa

import (
	"cmp"
	"fmt"
	"iter"
	"maps"
	"slices"
	"sort"
)

type (
	State int
	Class int
	Match int
)

const (
	NoState State = 0
	NoClass Class = 0
	NoMatch Match = 0
)

type Partition[S Symbol] struct {
	parts []Set[S]
}

func Split[S Symbol](sets []Set[S]) *Partition[S] {
	g := grouping[S]{seen: map[string]int{}}
	none := string(make([]byte, len(sets)))

	for _, a := range cutsOf(sets).atoms() {
		if sig := signature(sets, a.Lo); sig != none {
			g.add(sig, a)
		}
	}

	p := &Partition[S]{parts: []Set[S]{{}}}

	for _, rs := range g.held {
		p.parts = append(p.parts, Of(rs...))
	}

	return p
}

type cuts[S Symbol] struct {
	at  []S
	top bool
}

func cutsOf[S Symbol](sets []Set[S]) cuts[S] {
	in := map[S]bool{}
	var c cuts[S]

	for _, s := range sets {
		for _, x := range s.r {
			in[x.Lo] = true

			if x.Hi == top[S]() {
				c.top = true
			} else {
				in[x.Hi+1] = true
			}
		}
	}

	c.at = slices.Sorted(maps.Keys(in))

	return c
}

func (c cuts[S]) atoms() []Range[S] {
	var out []Range[S]

	for i := 1; i < len(c.at); i++ {
		out = append(out, Range[S]{c.at[i-1], c.at[i] - 1})
	}

	if c.top && len(c.at) > 0 {
		out = append(out, Range[S]{c.at[len(c.at)-1], top[S]()})
	}

	return out
}

func signature[S Symbol](sets []Set[S], x S) string {
	in := make([]byte, len(sets))

	for j, s := range sets {
		if s.Has(x) {
			in[j] = 1
		}
	}

	return string(in)
}

type grouping[S Symbol] struct {
	seen map[string]int
	held [][]Range[S]
}

func (g *grouping[S]) add(sig string, r Range[S]) {
	at, ok := g.seen[sig]

	if !ok {
		at = len(g.held)
		g.seen[sig] = at
		g.held = append(g.held, nil)
	}

	g.held[at] = append(g.held[at], r)
}

func (p *Partition[S]) Len() int { return len(p.parts) - 1 }

func (p *Partition[S]) Part(at Class) Set[S] { return p.parts[at] }

func (p *Partition[S]) All() iter.Seq2[Class, Set[S]] {
	return func(yield func(Class, Set[S]) bool) {
		for i := 1; i < len(p.parts); i++ {
			if !yield(Class(i), p.parts[i]) {
				return
			}
		}
	}
}

func (p *Partition[S]) Covers(s Set[S]) []Class {
	var out []Class

	for i := 1; i < len(p.parts); i++ {
		if !p.parts[i].Intersect(s).Empty() {
			out = append(out, Class(i))
		}
	}

	return out
}

func (p *Partition[S]) refine(sets []Set[S]) error {
	for _, set := range sets {
		var got Set[S]

		for _, at := range p.Covers(set) {
			got = got.Union(p.parts[at])
		}

		if !got.Equal(set) {
			return fmt.Errorf("the classes split %s, so they cannot read it", set)
		}
	}

	return nil
}

type Index[S Symbol] []Entry[S]

type Entry[S Symbol] struct {
	Range[S]
	Class Class
}

func (p *Partition[S]) Index() Index[S] {
	n := 0

	for _, s := range p.parts {
		n += len(s.r)
	}

	ix := make(Index[S], 0, n)

	for i, s := range p.parts {
		for _, x := range s.r {
			ix = append(ix, Entry[S]{x, Class(i)})
		}
	}

	slices.SortFunc(ix, func(a, b Entry[S]) int { return cmp.Compare(a.Lo, b.Lo) })

	return ix
}

func (ix Index[S]) Of(x S) Class {
	at := sort.Search(len(ix), func(i int) bool { return ix[i].Hi >= x })

	if at < len(ix) && ix[at].Lo <= x {
		return ix[at].Class
	}

	return NoClass
}
