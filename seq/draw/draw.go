package draw

import (
	"fmt"
	"strings"

	"github.com/cptaffe/automaton"
	"github.com/cptaffe/automaton/dot"
	"github.com/cptaffe/automaton/seq"
)

const base = 0xE000

func Draw(t *seq.Table) []byte { return dot.Draw(Machine(t)) }

func Machine(t *seq.Table) dot.Machine {
	sets := make([]automaton.Set, t.Alpha.Size())
	for s := range sets {
		sets[s] = automaton.One(rune(base + s))
	}
	c := automaton.Partition(sets)
	sym := make([]seq.Sym, c.Len())
	for i := range sym {
		sym[i] = seq.Dead
		if rs := c.Part(automaton.Class(i)).Ranges(); len(rs) == 1 && rs[0].Lo == rs[0].Hi && rs[0].Lo >= base {
			sym[i] = seq.Sym(rs[0].Lo - base)
		}
	}
	return machine{t: t, classes: c, sym: sym}
}

type machine struct {
	t       *seq.Table
	classes *automaton.Classes
	sym     []seq.Sym
}

func (m machine) Name() string { return m.t.Name }

func (m machine) Note() string {
	return fmt.Sprintf("%s -- %s, %s", m.t.Name, dot.Plural(len(m.t.Next), "state"), dot.Plural(int(m.t.Alpha.Size()), "symbol"))
}

func (m machine) States() int { return len(m.t.Next) }

func (m machine) Starts() map[string]automaton.State {
	out := map[string]automaton.State{}
	for scope, s := range m.t.Start {
		out[scope] = automaton.State(s)
	}
	return out
}

func (m machine) Classes() *automaton.Classes { return m.classes }

func (m machine) Next(s automaton.State, c automaton.Class) automaton.State {
	sym := m.sym[c]
	if sym == seq.Dead {
		return automaton.NoState
	}
	if n := m.t.Next[s][sym]; n != seq.Dead {
		return automaton.State(n)
	}
	return automaton.NoState
}

func (m machine) Accepts(s automaton.State) (string, bool) {
	a := m.t.Accept[s]
	if a == nil {
		return "", false
	}
	return Label(a), true
}

func (m machine) Named(s automaton.Set) (string, bool) {
	var syms []seq.Sym
	for _, r := range s.Ranges() {
		if r.Lo < base || r.Hi >= base+rune(m.t.Alpha.Size()) {
			return "", false
		}
		for c := r.Lo; c <= r.Hi; c++ {
			syms = append(syms, seq.Sym(c-base))
		}
	}
	var parts []string
	for i := 0; i < len(syms); {
		j := i
		for j+1 < len(syms) && syms[j+1] == syms[j]+1 && m.t.Alpha.IsKey(syms[j+1]) {
			j++
		}
		switch {
		case j-i >= 2:
			parts = append(parts, m.name(syms[i])+"-"+m.name(syms[j]))
		default:
			for k := i; k <= j; k++ {
				parts = append(parts, m.name(syms[k]))
			}
		}
		i = j + 1
	}
	return strings.Join(parts, " "), len(parts) > 0
}

func (m machine) name(s seq.Sym) string {
	if m.t.Alpha.IsKey(s) {
		return m.t.Alpha.Name(s)
	}
	return "<" + m.t.Alpha.Name(s) + ">"
}

func Label(a *seq.Accept) string {
	var parts []string
	if a.Action != "" {
		line, _, more := strings.Cut(strings.TrimSpace(a.Action), "\n")
		if more || len(line) > 24 {
			line = strings.TrimSpace(line[:min(len(line), 21)]) + "..."
		}
		parts = append(parts, line)
	}
	if a.When != nil {
		parts = append(parts, "when "+a.When.String())
	}
	if a.Goto != "" {
		parts = append(parts, "-> "+a.Goto)
	}
	if a.Hands != "" {
		parts = append(parts, "=> "+a.Hands)
	}
	return strings.Join(parts, " ")
}
