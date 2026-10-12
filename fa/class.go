package fa

import (
	"fmt"
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

type Classes[S Symbol] struct {
	parts []Set[S]
	by    []Range[S]
	at    []Class
}

func Partition[S Symbol](sets []Set[S]) *Classes[S] {
	cuts := map[S]bool{}
	reachesTop := false

	for _, s := range sets {
		for _, x := range s.r {
			cuts[x.Lo] = true

			if x.Hi == top[S]() {
				reachesTop = true
			} else {
				cuts[x.Hi+1] = true
			}
		}
	}

	edges := make([]S, 0, len(cuts))

	for r := range cuts {
		edges = append(edges, r)
	}

	sort.Slice(edges, func(i, j int) bool { return edges[i] < edges[j] })

	type atom struct {
		Range[S]
		sig string
	}

	var atoms []atom

	for i := range edges {
		var a atom

		switch {
		case i+1 < len(edges):
			a.Range = Range[S]{edges[i], edges[i+1] - 1}
		case reachesTop:
			a.Range = Range[S]{edges[i], top[S]()}
		default:
			continue
		}

		in := make([]byte, len(sets))

		for j, s := range sets {
			if s.Has(a.Lo) {
				in[j] = 1
			}
		}

		a.sig = string(in)
		atoms = append(atoms, a)
	}

	c := &Classes[S]{}
	seen := map[string]int{}
	none := string(make([]byte, len(sets)))

	var held [][]Range[S]

	for _, a := range atoms {
		if a.sig == none {
			continue
		}

		at, ok := seen[a.sig]

		if !ok {
			at = len(held)
			seen[a.sig] = at
			held = append(held, nil)
		}

		held[at] = append(held[at], a.Range)
	}

	c.parts = []Set[S]{{}}

	for _, rs := range held {
		c.parts = append(c.parts, Of(rs...))
	}

	for i, p := range c.parts {
		for _, x := range p.r {
			c.by = append(c.by, x)
			c.at = append(c.at, Class(i))
		}
	}

	order := make([]int, len(c.by))

	for i := range order {
		order[i] = i
	}

	sort.Slice(order, func(i, j int) bool { return c.by[order[i]].Lo < c.by[order[j]].Lo })
	by, at := make([]Range[S], len(order)), make([]Class, len(order))

	for i, o := range order {
		by[i], at[i] = c.by[o], c.at[o]
	}

	c.by, c.at = by, at

	return c
}

func (c *Classes[S]) Len() int { return len(c.parts) - 1 }

func (c *Classes[S]) Part(at Class) Set[S] { return c.parts[at] }

func (c *Classes[S]) Of(x S) Class {
	at := sort.Search(len(c.by), func(i int) bool { return c.by[i].Hi >= x })

	if at < len(c.by) && c.by[at].Lo <= x {
		return c.at[at]
	}

	return NoClass
}

func (c *Classes[S]) Covers(s Set[S]) []Class {
	var out []Class

	for i := 1; i < len(c.parts); i++ {
		if !c.parts[i].Intersect(s).Empty() {
			out = append(out, Class(i))
		}
	}

	return out
}

func (c *Classes[S]) refine(sets []Set[S]) error {
	for _, set := range sets {
		var got Set[S]

		for _, at := range c.Covers(set) {
			got = got.Union(c.parts[at])
		}

		if !got.Equal(set) {
			return fmt.Errorf("the classes split %s, so they cannot read it", set)
		}
	}

	return nil
}
