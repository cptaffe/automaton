package seq

import (
	"fmt"

	"github.com/cptaffe/automaton/fa"
)

type Machine struct {
	t     *Table
	scope int
	state int
}

func NewMachine(t *Table, scope string) (*Machine, error) {
	at := t.Scope(scope)
	if scope == "" && len(t.Scopes) > 0 {
		at = 1
	}
	if at == 0 {
		return nil, fmt.Errorf("%s has no scope %q", t.Name, scope)
	}
	return &Machine{t: t, scope: at}, nil
}

func (m *Machine) Scope() string { return m.t.Scopes[m.scope-1].Name }

func (m *Machine) start() int { return m.t.Scopes[m.scope-1].Start }

func (m *Machine) Pending() bool { return m.state != 0 }

func (m *Machine) Step(s Sym, env Env) (bool, []*Accept) {
	if m.t.Alpha.IsEvent(s) {
		return m.event(s, env)
	}
	k := m.t.Classes.Of(s)
	var fired []*Accept
	if m.state != 0 {
		if ok, f := m.from(m.state, k, env); ok {
			return true, f
		}
		fired = m.Settle(env)
	}
	ok, f := m.from(m.start(), k, env)
	return ok, append(fired, f...)
}

func (m *Machine) event(s Sym, env Env) (bool, []*Accept) {
	if st := &m.t.States[m.state]; st.Settles && s == st.Settle {
		return true, m.Settle(env)
	}
	k := m.t.Classes.Of(s)
	if m.state != 0 {
		return m.from(m.state, k, env)
	}
	return m.from(m.start(), k, env)
}

func (m *Machine) Settle(env Env) []*Accept {
	at := m.state
	m.state = 0
	if a := m.ready(at, env); a != nil {
		return m.fire(a)
	}
	return nil
}

func (m *Machine) Waits() []string {
	if m.state == 0 {
		return nil
	}
	var out []string
	st := &m.t.States[m.state]
	for s := Sym(m.t.Alpha.Keys()); int(s) < m.t.Alpha.Size(); s++ {
		k := m.t.Classes.Of(s)
		if st.Settles && s == st.Settle || st.Next[k] != 0 || len(st.Any[k]) > 0 {
			out = append(out, m.t.Alpha.Name(s))
		}
	}
	return out
}

func (m *Machine) from(at int, k fa.Class, env Env) (bool, []*Accept) {
	if n := m.t.States[at].Next[k]; n != 0 {
		ready, going := m.ready(n, env), m.going(n, env)
		switch {
		case ready != nil && (!going || !m.t.States[n].Settles):
			return true, m.fire(ready)
		case ready != nil || going:
			m.state = n
			return true, nil
		}
	}
	for _, a := range m.t.States[at].Any[k] {
		if Holds(a.When, env) {
			var fired []*Accept
			if at == m.state {
				fired = m.Settle(env)
			}
			return true, append(fired, m.fire(a)...)
		}
	}
	return false, nil
}

func (m *Machine) ready(s int, env Env) *Accept {
	for _, a := range m.t.States[s].Accept {
		if Holds(a.When, env) {
			return a
		}
	}
	return nil
}

func (m *Machine) going(s int, env Env) bool {
	for _, c := range m.t.States[s].Live {
		if Holds(c, env) {
			return true
		}
	}
	return false
}

func (m *Machine) fire(a *Accept) []*Accept {
	m.state = 0
	if a.Goto != 0 {
		m.scope = a.Goto
	}
	return []*Accept{a}
}
