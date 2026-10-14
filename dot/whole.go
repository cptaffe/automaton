package dot

import "fmt"

type box struct {
	m    Machine
	i    int
	skip map[int]bool
}

func Whole(ms []Machine, name string) Graph {
	g := Graph{
		Name:     name,
		Look:     "machine",
		Compound: true,
	}

	bs := make([]box, len(ms))

	for i, m := range ms {
		bs[i] = box{m: m, i: i, skip: kept(m)}
	}

	for _, b := range bs {
		g.Groups = append(g.Groups, b.group())
		g.Edges = append(g.Edges, b.edges()...)
	}

	for _, b := range bs {
		g.Edges = append(g.Edges, b.hands(bs)...)
	}

	return g
}

func (b box) id(s int) string { return fmt.Sprintf("%s%d", tag(b.i), s) }

func (b box) states() []int {
	var out []int

	for s := 1; s <= b.m.States(); s++ {
		if !b.skip[s] {
			out = append(out, s)
		}
	}

	return out
}

func (b box) group() Group {
	out := Group{ID: fmt.Sprint(b.i), Label: b.m.Name()}

	for _, s := range b.states() {
		out.Nodes = append(out.Nodes, state(b.id(s), s, b.m.Name(), s == start(b.m), b.m))
	}

	return out
}

func (b box) edges() []Edge {
	var out []Edge

	for _, s := range b.states() {
		for _, r := range runs(b.m, b.skip, s) {
			out = append(out, Edge{
				From:   b.id(s),
				To:     b.id(r.to),
				Label:  " " + times(r.move.Label, r.n) + " ",
				Class:  r.move.Class,
				Dashed: r.move.Dashed,
			})
		}
	}

	return out
}

func (b box) hands(bs []box) []Edge {
	leaves, ok := b.m.(Leaves)

	if !ok {
		return nil
	}

	var out []Edge

	for s := 1; s <= b.m.States(); s++ {
		scope, ok := leaves.Leaves(s)

		if !ok {
			continue
		}

		for _, to := range bs {
			if to.m.Name() == scope {
				out = append(out, Edge{
					From:   b.id(s),
					To:     to.id(start(to.m)),
					Class:  "hands",
					Dashed: true,
					Loose:  true,
				})
			}
		}
	}

	return out
}

func tag(i int) string { return fmt.Sprintf("m%d_s", i) }

func start(m Machine) int {
	if s := m.Starts(); len(s) > 0 {
		return s[0].State
	}

	return 0
}
