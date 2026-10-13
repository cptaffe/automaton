package hotkey

import (
	"os"
	"testing"

	"github.com/cptaffe/automaton/seq"
)

func TestTheExampleCompiles(t *testing.T) {
	f, err := os.Open("../../examples/hotkeys")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	r, err := Parse(f, "hotkeys", nil)
	if err != nil {
		t.Fatal(err)
	}
	m, err := seq.NewMachine(r.Table, "")
	if err != nil {
		t.Fatal(err)
	}
	acme := seq.Env{"app": "devdraw"}
	for _, c := range []struct {
		key   string
		fired string
	}{
		{"control-x", ""},
		{"control-s", "ctl put"},
		{"control-x", ""},
		{"j", ""},
		{"control-x", ""},
		{"settle", ""},
		{"command-w", "ctl del"},
	} {
		ok, fired := m.Step(sym(r.Table.Alpha, c.key), acme)
		if !ok || actions(fired) != c.fired || m.Pending() != (c.key == "control-x") {
			t.Errorf("%s took %v, fired %q and left pending %v; want %q", c.key, ok, actions(fired), m.Pending(), c.fired)
		}
	}
}
