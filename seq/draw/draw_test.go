package draw_test

import (
	"strings"
	"testing"

	"github.com/cptaffe/automaton/seq"
	"github.com/cptaffe/automaton/seq/draw"
	"github.com/cptaffe/automaton/seq/keypad"
)

func TestDraw(t *testing.T) {
	tree, err := keypad.Parse(strings.NewReader("%delay T 5s\n\nstart:\n  %settles T\n  0           do operator -> start\n  NXX         do short when on\n  960-X       do long\n"), "plan.tree")
	if err != nil {
		t.Fatal(err)
	}
	tab, err := keypad.Compile(tree)
	if err != nil {
		t.Fatal(err)
	}
	b, err := draw.Draw(tab)
	if err != nil {
		t.Fatal(err)
	}
	out := string(b)
	for _, want := range []string{"2-8", "0-9", "<T>", "operator -> start", "short when on"} {
		if !strings.Contains(out, want) {
			t.Errorf("the drawing lacks %q", want)
		}
	}
}

func TestLabel(t *testing.T) {
	long := &seq.Accept{Action: "echo a very long command that goes on and on"}
	if got := draw.Label(long); !strings.HasSuffix(got, "...") || len(got) > 24 {
		t.Errorf("label %q", got)
	}
	multi := &seq.Accept{Action: "a\nb", Goto: "next"}
	if got := draw.Label(multi); got != "a... -> next" {
		t.Errorf("label %q", got)
	}
}
