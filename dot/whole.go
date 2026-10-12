package dot

import "fmt"

func Whole(ms []Machine, name string) ([]byte, error) {
	g := Graph{
		Name:     name,
		Look:     "machine",
		Compound: true,
	}

	skips := make([]map[int]bool, len(ms))

	for i, m := range ms {
		skips[i] = kept(m)
	}

	for i, m := range ms {
		box := Group{ID: fmt.Sprint(i), Label: m.Name()}

		for s := 1; s <= m.States(); s++ {
			if skips[i][s] {
				continue
			}

			box.Nodes = append(box.Nodes, state(fmt.Sprintf("%s%d", tag(i), s), s, m.Name(), s == start(m), m))
		}

		g.Groups = append(g.Groups, box)

		for s := 1; s <= m.States(); s++ {
			if skips[i][s] {
				continue
			}

			for _, r := range runs(m, skips[i], s) {
				g.Edges = append(g.Edges, Edge{
					From:  fmt.Sprintf("%s%d", tag(i), s),
					To:    fmt.Sprintf("%s%d", tag(i), r.to),
					Label: " " + times(r.label, r.n) + " ",
				})
			}
		}
	}

	for i, m := range ms {
		leaves, ok := m.(Leaves)

		if !ok {
			continue
		}

		for s := 1; s <= m.States(); s++ {
			scope, ok := leaves.Leaves(s)

			if !ok {
				continue
			}

			for j, other := range ms {
				if other.Name() != scope {
					continue
				}

				g.Edges = append(g.Edges, Edge{
					From:   fmt.Sprintf("%s%d", tag(i), s),
					To:     fmt.Sprintf("%s%d", tag(j), start(other)),
					Class:  "hands",
					Dashed: true,
					Loose:  true,
				})
			}
		}
	}

	return g.Bytes()
}

func tag(i int) string { return fmt.Sprintf("m%d_s", i) }

func start(m Machine) int {
	if s := m.Starts(); len(s) > 0 {
		return s[0].State
	}

	return 0
}
