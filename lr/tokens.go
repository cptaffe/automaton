package lr

import (
	"sort"
	"strings"

	"github.com/cptaffe/automaton/fa"
)

type Tokens struct {
	dfa   *fa.DFA[int]
	words []string
	at    map[string]int
}

func (g *Grammar) Tokens() (*Tokens, error) { return tokensOf(g.Rules) }

func tokensOf(rs []Rule) (*Tokens, error) {
	v := vocabOf(rs)

	if len(v.words) == 0 {
		return nil, nil
	}

	g := &nest{vocab: v, spans: map[string]span{}}

	for _, r := range rs {
		g.spans[r.Name] = span{g.n.State(), g.n.State()}
	}

	for _, r := range rs {
		for _, alt := range r.Alts {
			g.alt(r.Name, alt)
		}
	}

	top := g.spans[rs[0].Name]
	g.n.Start(top.open)
	g.n.Accepts(top.shut, fa.Match(1))

	d, err := g.n.DFA()
	if err != nil {
		return nil, err
	}

	return &Tokens{dfa: d.Minimise(), words: v.words, at: v.at}, nil
}

func symbols(alt string) []string {
	var out []string

	for _, w := range strings.Fields(alt) {
		if w != "nothing" {
			out = append(out, w)
		}
	}

	return out
}

type vocab struct {
	rules map[string]bool
	words []string
	at    map[string]int
}

func vocabOf(rs []Rule) vocab {
	v := vocab{rules: map[string]bool{}, at: map[string]int{}}

	for _, r := range rs {
		v.rules[r.Name] = true
	}

	for _, r := range rs {
		for _, alt := range r.Alts {
			for _, w := range symbols(alt) {
				v.add(w)
			}
		}
	}

	return v
}

func (v *vocab) add(w string) {
	if _, seen := v.at[w]; !seen && !v.rules[w] {
		v.words = append(v.words, w)
		v.at[w] = len(v.words)
	}
}

type span struct{ open, shut int }

type nest struct {
	vocab
	n     fa.NFA[int]
	spans map[string]span
}

func (g *nest) alt(name, alt string) {
	here := g.spans[name].open

	for _, w := range symbols(alt) {
		next := g.n.State()

		if inner, ok := g.spans[w]; ok {
			g.n.Eps(here, inner.open)
			g.n.Eps(inner.shut, next)
		} else {
			g.n.On(here, fa.One(g.at[w]), next)
		}

		here = next
	}

	g.n.Eps(here, g.spans[name].shut)
}

func (t *Tokens) DFA() *fa.DFA[int] { return t.dfa }

func (t *Tokens) Words() []string { return append([]string(nil), t.words...) }

func (t *Tokens) Step(s fa.State, word string) fa.State {
	r, ok := t.at[word]

	if !ok {
		return fa.NoState
	}

	return t.dfa.Step(s, r)
}

func (t *Tokens) In(c fa.Class) []string {
	var out []string

	for _, x := range t.dfa.Classes().Part(c).Ranges() {
		for w := x.Lo; w <= x.Hi; w++ {
			out = append(out, t.words[w-1])
		}
	}

	sort.Strings(out)

	return out
}
