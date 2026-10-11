package draw

import (
	"fmt"
	"sort"
	"strings"

	"github.com/cptaffe/automaton/dot"
	"github.com/cptaffe/automaton/seq"
)

func Draw(t *seq.Table) ([]byte, error) { return dot.Draw(Machine(t)) }

func Machine(t *seq.Table) dot.Machine { return machine{t} }

type machine struct{ t *seq.Table }

func (m machine) Name() string { return m.t.Name }

func (m machine) Note() string {
	return fmt.Sprintf("%s -- %s, %s", m.t.Name, dot.Plural(m.States(), "state"), dot.Plural(m.t.Alpha.Size(), "symbol"))
}

func (m machine) States() int { return len(m.t.Next) - 1 }

func (m machine) Starts() []dot.Start {
	out := make([]dot.Start, len(m.t.Scopes))
	for i, scope := range m.t.Scopes {
		out[i] = dot.Start{Name: scope, State: m.t.Start[scope]}
	}
	return out
}

func (m machine) Moves(s int) []dot.Move {
	by := map[int][]seq.Sym{}
	var order []int
	for sym, to := range m.t.Next[s] {
		if to == 0 {
			continue
		}
		if _, ok := by[to]; !ok {
			order = append(order, to)
		}
		by[to] = append(by[to], seq.Sym(sym))
	}
	sort.Ints(order)
	out := make([]dot.Move, len(order))
	for i, to := range order {
		out[i] = dot.Move{To: to, Label: m.names(by[to])}
	}
	if settle := m.t.Settle[s]; settle != 0 {
		out = append(out, dot.Move{To: s, Label: m.name(settle), Dashed: true, Class: "settle"})
	}
	return out
}

func (m machine) Accepts(s int) (string, bool) {
	as := m.t.Accept[s]
	if len(as) == 0 {
		return "", false
	}
	labels := make([]string, len(as))
	for i, a := range as {
		labels[i] = Label(a)
	}
	return strings.Join(labels, "\n"), true
}

func (m machine) names(syms []seq.Sym) string {
	var parts []string
	for i := 0; i < len(syms); {
		j := i
		for j+1 < len(syms) && syms[j+1] == syms[j]+1 && m.t.Alpha.IsKey(syms[j+1]) {
			j++
		}
		if j-i >= 2 {
			parts = append(parts, m.name(syms[i])+"-"+m.name(syms[j]))
		} else {
			for k := i; k <= j; k++ {
				parts = append(parts, m.name(syms[k]))
			}
		}
		i = j + 1
	}
	return strings.Join(parts, " ")
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
	return strings.Join(parts, " ")
}
