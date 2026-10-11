package hotkey

import "github.com/cptaffe/automaton/seq"

type Matcher struct {
	r     *Rules
	state int
	mode  string
}

func (r *Rules) Matcher() *Matcher {
	return &Matcher{r: r, state: seq.Dead}
}

func (m *Matcher) Key(key string, env seq.Env) (bool, []*seq.Accept, []string) {
	t := m.r.Table
	sym, known := t.Alpha.Of(key)
	var fired []*seq.Accept
	if m.state != seq.Dead {
		if known {
			if n := t.Next[m.state][sym]; n != seq.Dead {
				return true, m.enter(n, env), m.waits()
			}
			if a := t.Any[m.state][sym]; a != nil {
				return true, m.fire(a, env), m.waits()
			}
		}
		fired = m.settle(env)
	}
	if !known {
		return false, fired, nil
	}
	for _, s := range m.starts(env) {
		if n := t.Next[s][sym]; n != seq.Dead {
			return true, append(fired, m.enter(n, env)...), m.waits()
		}
		if a := t.Any[s][sym]; a != nil {
			return true, append(fired, m.fire(a, env)...), m.waits()
		}
	}
	return false, fired, nil
}

func (m *Matcher) Event(name string, env seq.Env) ([]*seq.Accept, []string) {
	t := m.r.Table
	sym, ok := t.Alpha.Of(name)
	if !ok || m.state == seq.Dead {
		return nil, nil
	}
	if n := t.Next[m.state][sym]; n != seq.Dead {
		return m.enter(n, env), m.waits()
	}
	if a := t.Any[m.state][sym]; a != nil {
		return m.fire(a, env), m.waits()
	}
	return nil, m.waits()
}

func (m *Matcher) Mode() string { return m.mode }

func (m *Matcher) starts(env seq.Env) []int {
	t := m.r.Table
	if m.mode != "" {
		return []int{t.Start[m.mode]}
	}
	var out []int
	for _, s := range m.r.Grammar.Scopes {
		if c := m.r.Grammar.When[s]; c == nil || c.Eval(env) {
			out = append(out, t.Start[s])
		}
	}
	return out
}

func (m *Matcher) enter(n int, env seq.Env) []*seq.Accept {
	if a := m.r.Table.Accept[n]; a != nil {
		return m.fire(a, env)
	}
	m.state = n
	return nil
}

func (m *Matcher) fire(a *seq.Accept, env seq.Env) []*seq.Accept {
	m.state = seq.Dead
	if a.When != nil && !a.When.Eval(env) {
		return nil
	}
	if a.Goto != "" {
		m.mode = a.Goto
	}
	return []*seq.Accept{a}
}

func (m *Matcher) settle(env seq.Env) []*seq.Accept {
	t := m.r.Table
	at := m.state
	m.state = seq.Dead
	for sym := t.Alpha.Keys(); sym < t.Alpha.Size(); sym++ {
		if n := t.Next[at][sym]; n != seq.Dead && t.Accept[n] != nil {
			return m.fire(t.Accept[n], env)
		}
	}
	return nil
}

func (m *Matcher) waits() []string {
	t := m.r.Table
	if m.state == seq.Dead {
		return nil
	}
	var out []string
	for sym := t.Alpha.Keys(); sym < t.Alpha.Size(); sym++ {
		if t.Next[m.state][sym] != seq.Dead || t.Any[m.state][sym] != nil {
			out = append(out, t.Alpha.Name(sym))
		}
	}
	return out
}
