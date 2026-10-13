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
	if m.state != 0 {
		if ok, f := m.from(m.state, s, env); ok || len(f) > 0 {
			return ok, f
		}
		m.Reset()
	}
	return m.from(m.start(), s, env)
}

func (m *Machine) event(s Sym, env Env) (bool, []*Accept) {
	if m.state != 0 {
		return m.from(m.state, s, env)
	}
	return m.from(m.start(), s, env)
}

func (m *Machine) Reset() { m.state = 0 }

func (m *Machine) Waits() []string {
	if m.state == 0 {
		return nil
	}
	var out []string
	st := &m.t.States[m.state]
	for s := Sym(m.t.Alpha.Keys()); int(s) < m.t.Alpha.Size(); s++ {
		k := m.t.Classes.Of(s)
		if st.Next[k] != 0 || len(st.Any[k]) > 0 {
			out = append(out, m.t.Alpha.Name(s))
		}
	}
	return out
}

func (m *Machine) from(at int, s Sym, env Env) (bool, []*Accept) {
	k := m.t.Classes.Of(s)
	if n := m.t.States[at].Next[k]; n != 0 {
		ready, going := m.ready(n, env), m.going(n, env)
		switch {
		case ready != nil && ready.Ahead > 0:
			fired := m.fire(ready)
			taken, more := m.Step(s, env)
			return taken, append(fired, more...)
		case ready != nil:
			return true, m.fire(ready)
		case going:
			m.state = n
			return true, nil
		}
	}
	for _, a := range m.t.States[at].Any[k] {
		if Holds(a.When, env) {
			return true, m.fire(a)
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
