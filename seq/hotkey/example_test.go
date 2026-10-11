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
	m := r.Matcher()
	if _, fired, _ := m.Key("command-m", seq.Env{"app": "devdraw"}); len(fired) != 1 || m.Mode() != "marks" {
		t.Errorf("command-m fired %v into mode %q", fired, m.Mode())
	}
}
