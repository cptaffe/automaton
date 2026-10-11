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
	tree, err := Parse(f, "dial.tree")
	if err != nil {
		t.Fatal(err)
	}
	tab, err := Compile(tree)
	if err != nil {
		t.Fatal(err)
	}
	if len(tab.Conflicts) > 0 || tab.Scopes[0] != "dial.start" {
		t.Fatal(tab.Scopes, tab.Conflicts)
	}
}
