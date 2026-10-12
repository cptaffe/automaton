package seq

import "fmt"

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
	var fired []*Accept
	if m.state != 0 {
		if ok, f := m.from(m.state, s, env); ok {
			return true, f
		}
		fired = m.Settle(env)
	}
	ok, f := m.from(m.start(), s, env)
	return ok, append(fired, f...)
}

func (m *Machine) event(s Sym, env Env) (bool, []*Accept) {
	if st := &m.t.States[m.state]; st.Settles && s == st.Settle {
		return true, m.Settle(env)
	}
	if m.state != 0 {
		return m.from(m.state, s, env)
	}
	return m.from(m.start(), s, env)
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
		if st.Settles && s == st.Settle || st.Next[s] != 0 || len(st.Any[s]) > 0 {
			out = append(out, m.t.Alpha.Name(s))
		}
	}
	return out
}

func (m *Machine) from(at int, s Sym, env Env) (bool, []*Accept) {
	if n := m.t.States[at].Next[s]; n != 0 {
		ready, going := m.ready(n, env), m.going(n, env)
		switch {
		case ready != nil && (!going || !m.t.States[n].Settles):
			return true, m.fire(ready)
		case ready != nil || going:
			m.state = n
			return true, nil
		}
	}
	for _, a := range m.t.States[at].Any[s] {
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
