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
	index []indexed[S]
}

type indexed[S Symbol] struct {
	Range[S]
	class Class
}

func Partition[S Symbol](sets []Set[S]) *Classes[S] {
	g := grouping[S]{seen: map[string]int{}}
	none := string(make([]byte, len(sets)))

	for _, a := range cutsOf(sets).atoms() {
		if sig := signature(sets, a.Lo); sig != none {
			g.add(sig, a)
		}
	}

	c := &Classes[S]{parts: []Set[S]{{}}}

	for _, rs := range g.held {
		c.parts = append(c.parts, Of(rs...))
	}

	c.reindex()

	return c
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

	c.at = make([]S, 0, len(in))

	for r := range in {
		c.at = append(c.at, r)
	}

	sort.Slice(c.at, func(i, j int) bool { return c.at[i] < c.at[j] })

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

func (c *Classes[S]) reindex() {
	for i, p := range c.parts {
		for _, x := range p.r {
			c.index = append(c.index, indexed[S]{x, Class(i)})
		}
	}

	sort.Slice(c.index, func(i, j int) bool { return c.index[i].Lo < c.index[j].Lo })
}

func (c *Classes[S]) Len() int { return len(c.parts) - 1 }

func (c *Classes[S]) Part(at Class) Set[S] { return c.parts[at] }

func (c *Classes[S]) Of(x S) Class {
	at := sort.Search(len(c.index), func(i int) bool { return c.index[i].Hi >= x })

	if at < len(c.index) && c.index[at].Lo <= x {
		return c.index[at].class
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
