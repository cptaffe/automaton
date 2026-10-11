package lr

import (
	"fmt"
	"sort"
	"strings"

	"github.com/cptaffe/automaton"
)

const first = rune(0xE000)

type Tokens struct {
	name string
	note string
	dfa  *automaton.DFA
	of   map[rune]string

	symbols int
}

func (g *Grammar) Tokens(name string) *Tokens { return tokensOf(g.Rules, name) }

func tokensOf(rs []Rule, name string) *Tokens {
	is := map[string]bool{}

	for _, r := range rs {
		is[r.Name] = true
	}

	at := map[string]rune{}
	of := map[rune]string{}

	for _, r := range rs {
		for _, alt := range r.Alts {
			for _, w := range strings.Fields(alt) {
				if is[w] || w == "nothing" {
					continue
				}

				if _, seen := at[w]; !seen {
					r := first + rune(len(at))
					at[w], of[r] = r, w
				}
			}
		}
	}

	if len(at) == 0 {
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
	n.Accepts(shut[rs[0].Name], automaton.Match(0))

	d := n.Determinise(automaton.Partition(n.Sets())).Minimise()

	return &Tokens{name: name, dfa: d, of: of, symbols: len(at)}
}

func (t *Tokens) SetNote(s string)            { t.note = s }
func (t *Tokens) Symbols() int                { return t.symbols }
func (t *Tokens) Name() string                { return t.name }
func (t *Tokens) Note() string                { return t.note }
func (t *Tokens) States() int                 { return t.dfa.States() }
func (t *Tokens) Classes() *automaton.Classes { return t.dfa.Classes() }
func (t *Tokens) Starts() map[string]automaton.State {
	return map[string]automaton.State{t.name: t.dfa.Start()}
}

func (t *Tokens) Next(s automaton.State, c automaton.Class) automaton.State {
	return t.dfa.Next(s, c)
}

func (t *Tokens) Accepts(s automaton.State) (string, bool) {
	return "", t.dfa.Accept(s) != automaton.NoMatch
}

func (t *Tokens) Named(s automaton.Set) (string, bool) {
	var out []string

	for _, r := range s.Ranges() {
		for c := r.Lo; c <= r.Hi; c++ {
			name, ok := t.of[c]

			if !ok {
				return "", false
			}

			out = append(out, name)
		}
	}

	if len(out) == 0 {
		return "", false
	}

	sort.Strings(out)

	if len(out) > 4 {
		return fmt.Sprintf("%s and %d more", strings.Join(out[:4], " "), len(out)-4), true
	}

	return strings.Join(out, " "), true
}
