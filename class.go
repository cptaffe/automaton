package automaton

import "sort"

type (
	State int
	Class int
	Match int
)

const (
	NoState State = -1
	NoClass Class = -1
	NoMatch Match = -1
)

type Classes struct {
	parts []Set
	by    []Range
	at    []Class
}

func Partition(sets []Set) *Classes {
	cuts := map[rune]bool{}

	for _, s := range sets {
		for _, x := range s.Ranges() {
			cuts[x.Lo] = true
			cuts[x.Hi+1] = true
		}
	}

	edges := make([]rune, 0, len(cuts))

	for r := range cuts {
		edges = append(edges, r)
	}

	sort.Slice(edges, func(i, j int) bool { return edges[i] < edges[j] })

	type atom struct {
		Range
		sig string
	}

	var atoms []atom

	for i := 0; i+1 <= len(edges)-1; i++ {
		a := atom{Range: Range{edges[i], edges[i+1] - 1}}

		if a.Lo > a.Hi {
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

	c := &Classes{}
	seen := map[string]int{}
	none := string(make([]byte, len(sets)))

	var held [][]Range

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

	for _, rs := range held {
		c.parts = append(c.parts, Of(rs...))
	}

	for i, p := range c.parts {
		for _, x := range p.Ranges() {
			c.by = append(c.by, x)
			c.at = append(c.at, Class(i))
		}
	}

	order := make([]int, len(c.by))

	for i := range order {
		order[i] = i
	}

	sort.Slice(order, func(i, j int) bool { return c.by[order[i]].Lo < c.by[order[j]].Lo })
	by, at := make([]Range, len(order)), make([]Class, len(order))

	for i, o := range order {
		by[i], at[i] = c.by[o], c.at[o]
	}

	c.by, c.at = by, at

	return c
}

func (c *Classes) Len() int { return len(c.parts) }

func (c *Classes) Parts() []Set { return c.parts }

func (c *Classes) Part(at Class) Set { return c.parts[at] }

func (c *Classes) Of(r rune) Class {
	at := sort.Search(len(c.by), func(i int) bool { return c.by[i].Hi >= r })

	if at < len(c.by) && c.by[at].Lo <= r {
		return c.at[at]
	}

	return NoClass
}

func (c *Classes) Covers(s Set) []Class {
	var out []Class

	for i, p := range c.parts {
		if !p.Intersect(s).Empty() {
			out = append(out, Class(i))
		}
	}

	return out
}
