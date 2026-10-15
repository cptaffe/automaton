package draw

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/cptaffe/automaton/dot"
	"github.com/cptaffe/automaton/fa"
	"github.com/cptaffe/automaton/seq"
)

func Draw(t *seq.Table) dot.Graph { return dot.Draw(Machine(t)) }

func Machine(t *seq.Table) dot.Machine { return machine{t} }

type machine struct{ t *seq.Table }

func (m machine) Name() string { return m.t.Name }

func (m machine) Note() string {
	return fmt.Sprintf("%s -- %s, %s", m.t.Name, dot.Plural(m.States(), "state"), dot.Plural(m.t.Partition.Len(), "class"))
}

func (m machine) States() int { return len(m.t.States) - 1 }

func (m machine) Starts() []dot.Start {
	out := make([]dot.Start, len(m.t.Scopes))
	for i, scope := range m.t.Scopes {
		out[i] = dot.Start{Name: scope.Name, State: scope.Start}
	}
	return out
}

type targets struct {
	syms map[int][]seq.Sym
	rest int
}

func (m machine) Moves(s int) []dot.Move {
	st := &m.t.States[s]
	t := m.targets(st)
	gotos := m.gotos(st)
	order := slices.Sorted(maps.Keys(t.syms))
	out := make([]dot.Move, len(order), len(order)+len(gotos))
	for i, to := range order {
		out[i] = m.move(t, to)
		if g := m.guard(to); g != "" && g != m.entered(s) {
			out[i].Label += "\nwhen " + g
		}
	}
	return append(out, gotos...)
}

func (m machine) targets(st *seq.State) targets {
	t := targets{syms: map[int][]seq.Sym{}}
	for k, to := range st.Next {
		if to == 0 {
			continue
		}
		if syms := m.syms(fa.Class(k)); len(syms) > 0 {
			t.syms[to] = append(t.syms[to], syms...)
		}
	}
	for _, syms := range t.syms {
		slices.Sort(syms)
	}
	t.rest = m.rest(t.syms)
	return t
}

func (m machine) syms(c fa.Class) []seq.Sym {
	var out []seq.Sym
	for _, r := range m.t.Partition.Part(c).Ranges() {
		for sym := r.Lo; sym <= r.Hi; sym++ {
			out = append(out, sym)
		}
	}
	return out
}

func (m machine) move(t targets, to int) dot.Move {
	mv := dot.Move{To: to, Label: m.names(t.syms[to])}
	if to == t.rest {
		mv.Label = "any"
	}
	if !slices.ContainsFunc(t.syms[to], m.t.Alpha.IsKey) {
		mv.Dashed, mv.Class = true, "event"
	}
	return mv
}

func (m machine) gotos(st *seq.State) []dot.Move {
	var out []dot.Move
	gone := map[int]bool{}
	for _, a := range st.Accept {
		if a.Goto == 0 {
			continue
		}
		if to := m.t.Scopes[a.Goto-1].Start; !gone[to] {
			gone[to] = true
			out = append(out, dot.Move{To: to, Dashed: true, Class: "goto"})
		}
	}
	return out
}

func (m machine) Accepts(s int) (string, bool) {
	as := m.t.States[s].Accept
	if len(as) == 0 {
		return "", false
	}
	g := m.guard(s)
	labels := make([]string, len(as))
	for i, a := range as {
		if a.When != nil && a.When.String() == g {
			a = &seq.Accept{Action: a.Action}
		}
		labels[i] = Label(a)
	}
	return strings.Join(labels, "\n"), true
}

func (m machine) entered(s int) string {
	for _, scope := range m.t.Scopes {
		if scope.Start == s {
			return ""
		}
	}
	return m.guard(s)
}

func (m machine) guard(s int) string {
	st := &m.t.States[s]
	cs := make([]seq.Cond, 0, len(st.Live)+len(st.Accept))
	cs = append(cs, st.Live...)
	for _, a := range st.Accept {
		cs = append(cs, a.When)
	}
	var or seq.Or
	seen := map[string]bool{}
	for _, c := range cs {
		if c == nil {
			return ""
		}
		if !seen[c.String()] {
			seen[c.String()] = true
			or = append(or, c)
		}
	}
	switch len(or) {
	case 0:
		return ""
	case 1:
		return or[0].String()
	}
	return or.String()
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
		parts = append(parts, wrap("when "+a.When.String(), 14))
	}
	return strings.Join(parts, "\n")
}

func wrap(s string, width int) string {
	var lines []string
	line := ""
	for _, w := range strings.Fields(s) {
		if line != "" && len(line)+1+len(w) > width {
			lines = append(lines, line)
			line = ""
		}
		if line != "" {
			line += " "
		}
		line += w
	}
	return strings.Join(append(lines, line), "\n")
}

func (m machine) rest(by map[int][]seq.Sym) int {
	keys, most, rest := 0, 0, 0
	for to, syms := range by {
		n := 0
		for _, s := range syms {
			if m.t.Alpha.IsKey(s) {
				n++
			}
		}
		keys += n
		if n > most || n == most && to < rest {
			most, rest = n, to
		}
	}
	if keys < m.t.Alpha.Keys() || most <= m.t.Alpha.Keys()/2 {
		return 0
	}
	return rest
}
