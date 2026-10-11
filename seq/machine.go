package seq

type Machine struct {
	t     *Table
	state int
	scope string
}

func NewMachine(t *Table, scope string) *Machine {
	return &Machine{t: t, state: Dead, scope: scope}
}

func (m *Machine) Scope() string { return m.scope }

func (m *Machine) Active() bool { return m.state != Dead }

func (m *Machine) Step(s Sym, env Env) (bool, []*Accept) {
	if m.state != Dead {
		return m.from(m.state, s, env)
	}
	if m.scope == "" {
		return false, nil
	}
	return m.from(m.t.Start[m.scope], s, env)
}

func (m *Machine) StepFrom(scope string, s Sym, env Env) (bool, []*Accept) {
	return m.from(m.t.Start[scope], s, env)
}

func (m *Machine) Settle(env Env) []*Accept {
	at := m.state
	m.state = Dead
	if at == Dead {
		return nil
	}
	for s := m.t.Alpha.Keys(); s < m.t.Alpha.Size(); s++ {
		if n := m.t.Next[at][s]; n != Dead && m.t.Accept[n] != nil {
			return m.fire(m.t.Accept[n], env)
		}
	}
	return nil
}

func (m *Machine) Waits() []string {
	if m.state == Dead {
		return nil
	}
	var out []string
	for s := m.t.Alpha.Keys(); s < m.t.Alpha.Size(); s++ {
		if m.t.Next[m.state][s] != Dead || m.t.Any[m.state][s] != nil {
			out = append(out, m.t.Alpha.Name(s))
		}
	}
	return out
}

func (m *Machine) from(at int, s Sym, env Env) (bool, []*Accept) {
	if n := m.t.Next[at][s]; n != Dead {
		if a := m.t.Accept[n]; a != nil {
			return true, m.fire(a, env)
		}
		m.state = n
		return true, nil
	}
	if a := m.t.Any[at][s]; a != nil {
		return true, m.fire(a, env)
	}
	return false, nil
}

func (m *Machine) fire(a *Accept, env Env) []*Accept {
	m.state = Dead
	if a.When != nil && !a.When.Eval(env) {
		return nil
	}
	if a.Goto != "" {
		m.scope = a.Goto
	}
	return []*Accept{a}
}
