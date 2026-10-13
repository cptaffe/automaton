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
		scope string
	}{{"command-g", "group"}, {"command-s", "group"}, {"escape", "main"}} {
		if ok, _ := m.Step(sym(r.Table.Alpha, c.key), acme); !ok || m.Scope() != c.scope {
			t.Errorf("%s took %v into %s; want %s", c.key, ok, m.Scope(), c.scope)
		}
	}
}
