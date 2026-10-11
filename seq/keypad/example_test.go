package keypad

import (
	"os"
	"testing"

	"github.com/cptaffe/automaton/seq"
)

func TestTheExampleCompiles(t *testing.T) {
	f, err := os.Open("../../examples/dial.tree")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	g, err := Parse(f, "dial.tree")
	if err != nil {
		t.Fatal(err)
	}
	tab, err := seq.Compile(g, Alphabet(g))
	if err != nil || len(tab.Conflicts) > 0 {
		t.Fatal(err, tab.Conflicts)
	}
}
