package dot

import "strings"

type loops struct {
	group map[string]int

	embeds map[int]bool
}

func find(rs []Rule, is map[string]bool) loops {
	out := loops{group: map[string]int{}, embeds: map[int]bool{}}

	refs := map[string][]string{}

	for _, r := range rs {
		seen := map[string]bool{}

		for _, alt := range r.Alts {
			for _, w := range word.FindAllString(alt, -1) {
				if is[w] && !seen[w] {
					seen[w] = true
					refs[r.Name] = append(refs[r.Name], w)
				}
			}
		}
	}

	var at, n int
	idx, low := map[string]int{}, map[string]int{}
	var stack []string
	on := map[string]bool{}

	type frame struct {
		name string
		next int
	}

	for _, r := range rs {
		if _, ok := idx[r.Name]; ok {
			continue
		}

		work := []frame{{name: r.Name}}
		idx[r.Name], low[r.Name] = at, at
		at++
		stack = append(stack, r.Name)
		on[r.Name] = true

		for len(work) > 0 {
			f := &work[len(work)-1]
			down := false

			for f.next < len(refs[f.name]) {
				w := refs[f.name][f.next]
				f.next++

				if _, ok := idx[w]; !ok {
					idx[w], low[w] = at, at
					at++
					stack = append(stack, w)
					on[w] = true
					work = append(work, frame{name: w})
					down = true

					break
				} else if on[w] {
					low[f.name] = min(low[f.name], idx[w])
				}
			}

			if down {
				continue
			}

			name := f.name
			work = work[:len(work)-1]

			if low[name] == idx[name] {
				var comp []string

				for {
					w := stack[len(stack)-1]
					stack = stack[:len(stack)-1]
					on[w] = false
					comp = append(comp, w)

					if w == name {
						break
					}
				}

				if len(comp) == 1 {
					self := false

					for _, w := range refs[comp[0]] {
						self = self || w == comp[0]
					}

					if !self {
						continue
					}
				}

				n++

				for _, w := range comp {
					out.group[w] = n
				}
			}

			if len(work) > 0 {
				up := work[len(work)-1].name
				low[up] = min(low[up], low[name])
			}
		}
	}

	out.nest(rs, is)

	return out
}

func (l loops) nest(rs []Rule, is map[string]bool) {
	for _, r := range rs {
		g := l.group[r.Name]

		if g == 0 {
			continue
		}

		for _, alt := range r.Alts {
			ws := strings.Fields(alt)
			var at []int

			for i, w := range ws {
				if is[w] && l.group[w] == g {
					at = append(at, i)
				}
			}

			if len(at) > 1 || (len(at) == 1 && at[0] > 0 && at[0] < len(ws)-1) {
				l.embeds[g] = true
			}
		}
	}
}

func Nested(rs []Rule) []string {
	is := map[string]bool{}

	for _, r := range rs {
		is[r.Name] = true
	}

	loop := find(rs, is)

	var out []string

	for _, r := range rs {
		if at := loop.group[r.Name]; at != 0 && loop.embeds[at] {
			out = append(out, r.Name)
		}
	}

	return out
}
