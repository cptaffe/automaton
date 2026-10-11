package lr

import (
	"sort"
	"strings"

	"github.com/cptaffe/automaton"
)

type Tokens struct {
	dfa   *automaton.DFA
	words []string
	at    map[string]rune
}

func (g *Grammar) Tokens() *Tokens { return tokensOf(g.Rules) }

func tokensOf(rs []Rule) *Tokens {
	is := map[string]bool{}

	for _, r := range rs {
		is[r.Name] = true
	}

	at := map[string]rune{}
	var words []string

	for _, r := range rs {
		for _, alt := range r.Alts {
			for _, w := range strings.Fields(alt) {
				if is[w] || w == "nothing" {
					continue
				}

				if _, seen := at[w]; !seen {
					words = append(words, w)
					at[w] = rune(len(words))
				}
			}
		}
	}

	if len(words) == 0 {
		return nil
	}

	var n automaton.NFA

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
					n.On(here, automaton.One(at[w]), next)
				}

				here = next
			}

			n.Eps(here, shut[r.Name])
		}
	}

	n.Start(open[rs[0].Name])
	n.Accepts(shut[rs[0].Name], automaton.Match(1))

	d := n.DFA().Minimise()

	return &Tokens{dfa: d, words: words, at: at}
}

func (t *Tokens) DFA() *automaton.DFA { return t.dfa }

func (t *Tokens) Words() []string { return append([]string(nil), t.words...) }

func (t *Tokens) Step(s automaton.State, word string) automaton.State {
	r, ok := t.at[word]

	if !ok {
		return automaton.NoState
	}

	return t.dfa.Step(s, r)
}

func (t *Tokens) In(c automaton.Class) []string {
	var out []string

	for _, x := range t.dfa.Classes().Part(c).Ranges() {
		for r := x.Lo; r <= x.Hi; r++ {
			out = append(out, t.words[r-1])
		}
	}

	sort.Strings(out)

	return out
}
