package keypad

import (
	"os"
	"testing"
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
	tab, err := Compile(g)
	if err != nil {
		t.Fatal(err)
	}
	if len(tab.Conflicts) > 0 || tab.Scopes[0].Name != "start" {
		t.Fatal(tab.Scopes, tab.Conflicts)
	}
}
