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
		if s != 0 {
			if ok, f := m.from(m.state, s, env); ok {
				return true, f
			}
		}
		fired = m.Settle(env)
	}
	if s == 0 {
		return false, fired
	}
	ok, f := m.from(m.start(), s, env)
	return ok, append(fired, f...)
}

func (m *Machine) event(s Sym, env Env) (bool, []*Accept) {
	switch {
	case m.state != 0 && s == m.t.Settle[m.state]:
		return true, m.Settle(env)
	case m.state != 0:
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
	for s := Sym(m.t.Alpha.Keys() + 1); int(s) <= m.t.Alpha.Size(); s++ {
		if m.t.Next[m.state][s] != 0 || len(m.t.Any[m.state][s]) > 0 || m.t.Settle[m.state] == s {
			out = append(out, m.t.Alpha.Name(s))
		}
	}
	return out
}

func (m *Machine) from(at int, s Sym, env Env) (bool, []*Accept) {
	if n := m.t.Next[at][s]; n != 0 {
		ready, going := m.ready(n, env), m.going(n, env)
		switch {
		case ready != nil && (!going || m.t.Settle[n] == 0):
			return true, m.fire(ready)
		case ready != nil || going:
			m.state = n
			return true, nil
		}
	}
	for _, a := range m.t.Any[at][s] {
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
	for _, a := range m.t.Accept[s] {
		if Holds(a.When, env) {
			return a
		}
	}
	return nil
}

func (m *Machine) going(s int, env Env) bool {
	for _, c := range m.t.Live[s] {
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
