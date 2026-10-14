package seqtest

import (
	"strings"
	"testing"

	"github.com/cptaffe/automaton/seq"
)

type Step struct {
	Key     string
	Env     seq.Env
	Taken   bool
	Fired   string
	Pending bool
}

func Machine(t testing.TB, tab *seq.Table, scope string) *seq.Machine {
	t.Helper()
	m, err := seq.NewMachine(tab, scope)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func Press(m *seq.Machine, a *seq.Alphabet, key string, env seq.Env) (bool, string) {
	s, ok := a.Of(key)
	if !ok {
		m.Reset()
		return false, ""
	}
	taken, fired := m.Step(s, env)
	return taken, Actions(fired)
}

func Actions(as []*seq.Accept) string {
	out := make([]string, 0, len(as))
	for _, a := range as {
		out = append(out, a.Action)
	}
	return strings.Join(out, ",")
}

func Prefix(keys string) []Step {
	fields := strings.Fields(keys)
	out := make([]Step, 0, len(fields))
	for _, k := range fields {
		out = append(out, Step{Key: k, Taken: true, Pending: true})
	}
	return out
}

func Run(t testing.TB, m *seq.Machine, a *seq.Alphabet, env seq.Env, steps []Step) {
	t.Helper()
	for i, want := range steps {
		in := env
		if want.Env != nil {
			in = want.Env
		}
		taken, fired := Press(m, a, want.Key, in)
		if taken != want.Taken || fired != want.Fired || m.Pending() != want.Pending {
			t.Fatalf("step %d, %s: taken %v, fired %q, pending %v; want taken %v, fired %q, pending %v",
				i+1, want.Key, taken, fired, m.Pending(), want.Taken, want.Fired, want.Pending)
			return
		}
	}
}
