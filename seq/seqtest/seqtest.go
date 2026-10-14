package seqtest

import (
	"strings"
	"testing"

	"github.com/cptaffe/automaton/seq"
)

type Step struct {
	Key     string
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

func Run(t testing.TB, m *seq.Machine, a *seq.Alphabet, env seq.Env, steps []Step) {
	t.Helper()
	for i, want := range steps {
		taken, fired := Press(m, a, want.Key, env)
		got := Step{Key: want.Key, Taken: taken, Fired: fired, Pending: m.Pending()}
		if got != want {
			t.Fatalf("step %d: got %+v; want %+v", i+1, got, want)
		}
	}
}
