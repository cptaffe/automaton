package hotkey

import "github.com/cptaffe/automaton/seq"

type Matcher struct {
	r *Rules
	m *seq.Machine
}

func (r *Rules) Matcher() *Matcher {
	return &Matcher{r: r, m: seq.NewMachine(r.Table, "")}
}

func (m *Matcher) Mode() string { return m.m.Scope() }

func (m *Matcher) Key(key string, env seq.Env) (bool, []*seq.Accept, []string) {
	sym, known := m.r.Table.Alpha.Of(key)
	var fired []*seq.Accept
	if m.m.Active() {
		if known {
			if ok, f := m.m.Step(sym, env); ok {
				return true, f, m.m.Waits()
			}
		}
		fired = m.m.Settle(env)
	}
	if !known {
		return false, fired, nil
	}
	if m.m.Scope() != "" {
		ok, f := m.m.Step(sym, env)
		return ok, append(fired, f...), m.m.Waits()
	}
	for _, s := range m.r.Grammar.Scopes {
		if c := m.r.When[s]; c != nil && !c.Eval(env) {
			continue
		}
		if ok, f := m.m.StepFrom(s, sym, env); ok {
			return true, append(fired, f...), m.m.Waits()
		}
	}
	return false, fired, nil
}

func (m *Matcher) Event(name string, env seq.Env) ([]*seq.Accept, []string) {
	sym, ok := m.r.Table.Alpha.Of(name)
	if !ok || !m.m.Active() {
		return nil, nil
	}
	_, fired := m.m.Step(sym, env)
	return fired, m.m.Waits()
}
