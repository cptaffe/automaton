package dot

import (
	"context"
	"slices"
	"strings"

	"github.com/cptaffe/automaton/dot/internal/scan"
	"github.com/cptaffe/automaton/lex"
)

type loops struct {
	group map[string]int

	embeds map[int]bool
}

type grammar struct {
	rules []Rule

	is map[string]bool

	refs map[string][]string
}

func read(rs []Rule) grammar {
	gr := grammar{rules: rs, is: map[string]bool{}, refs: map[string][]string{}}

	for _, r := range rs {
		gr.is[r.Name] = true
	}

	for _, r := range rs {
		seen := map[string]bool{}

		for _, alt := range r.Alts {
			for _, w := range tokens(alt) {
				if gr.is[w] && !seen[w] {
					seen[w] = true
					gr.refs[r.Name] = append(gr.refs[r.Name], w)
				}
			}
		}
	}

	return gr
}

func tokens(alt string) []string {
	toks, err := lex.Lex(context.Background(), scan.Spec, strings.NewReader(alt), "start")

	if err != nil {
		return nil
	}

	var out []string

	for t := range toks {
		out = append(out, t.Text)
	}

	return out
}

func (gr grammar) cycle(comp []string) bool {
	return len(comp) > 1 || slices.Contains(gr.refs[comp[0]], comp[0])
}

type tarjan struct {
	refs map[string][]string

	idx, low map[string]int

	stack []string
	on    map[string]bool

	comps [][]string
}

func (gr grammar) components() [][]string {
	t := tarjan{refs: gr.refs, idx: map[string]int{}, low: map[string]int{}, on: map[string]bool{}}

	for _, r := range gr.rules {
		if _, ok := t.idx[r.Name]; !ok {
			t.visit(r.Name)
		}
	}

	return t.comps
}

func (t *tarjan) visit(name string) {
	at := len(t.idx)
	t.idx[name], t.low[name] = at, at
	t.stack = append(t.stack, name)
	t.on[name] = true

	for _, w := range t.refs[name] {
		if _, ok := t.idx[w]; !ok {
			t.visit(w)
			t.low[name] = min(t.low[name], t.low[w])
		} else if t.on[w] {
			t.low[name] = min(t.low[name], t.idx[w])
		}
	}

	if t.low[name] == t.idx[name] {
		t.comps = append(t.comps, t.pop(name))
	}
}

func (t *tarjan) pop(name string) []string {
	var comp []string

	for {
		w := t.stack[len(t.stack)-1]
		t.stack = t.stack[:len(t.stack)-1]
		t.on[w] = false
		comp = append(comp, w)

		if w == name {
			return comp
		}
	}
}

func find(gr grammar) loops {
	out := loops{group: map[string]int{}, embeds: map[int]bool{}}

	var n int

	for _, comp := range gr.components() {
		if !gr.cycle(comp) {
			continue
		}

		n++

		for _, w := range comp {
			out.group[w] = n
		}
	}

	out.nest(gr)

	return out
}

func (l loops) nest(gr grammar) {
	for _, r := range gr.rules {
		g := l.group[r.Name]

		if g == 0 {
			continue
		}

		for _, alt := range r.Alts {
			ws := tokens(alt)
			var at []int

			for i, w := range ws {
				if gr.is[w] && l.group[w] == g {
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
	loop := find(read(rs))

	var out []string

	for _, r := range rs {
		if at := loop.group[r.Name]; at != 0 && loop.embeds[at] {
			out = append(out, r.Name)
		}
	}

	return out
}
