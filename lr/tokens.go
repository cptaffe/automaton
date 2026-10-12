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

func (g *Grammar) Tokens() *Tokens { return tokensOf(g.Rules) }

func tokensOf(rs []Rule) *Tokens {
	is := map[string]bool{}

	for _, r := range rs {
		is[r.Name] = true
	}

	at := map[string]int{}
	var words []string

	for _, r := range rs {
		for _, alt := range r.Alts {
			for _, w := range strings.Fields(alt) {
				if is[w] || w == "nothing" {
					continue
				}

				if _, seen := at[w]; !seen {
					words = append(words, w)
					at[w] = len(words)
				}
			}
		}
	}

	if len(words) == 0 {
		return nil
	}

	var n fa.NFA[int]

	open := map[string]int{}
	shut := map[string]int{}

	for _, r := range rs {
		open[r.Name], shut[r.Name] = n.State(), n.State()
	}

	for _, r := range rs {
		for _, alt := range r.Alts {
			ws := strings.Fields(alt)
			here := open[r.Name]

			for _, w := range ws {
				if w == "nothing" {
					continue
				}

				next := n.State()

				if is[w] {

					n.Eps(here, open[w])
					n.Eps(shut[w], next)
				} else {
					n.On(here, fa.One(at[w]), next)
				}

				here = next
			}

			n.Eps(here, shut[r.Name])
		}
	}

	n.Start(open[rs[0].Name])
	n.Accepts(shut[rs[0].Name], fa.Match(1))

	d := n.DFA().Minimise()

	return &Tokens{dfa: d, words: words, at: at}
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
