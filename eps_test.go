package automaton_test

import (
	"strings"
	"testing"

	"github.com/cptaffe/automaton"
)

func TestTheTableHasNoEmptyTransitions(t *testing.T) {
	d, err := automaton.Build([]automaton.Rule[rune]{
		{Name: "word", Expr: automaton.Star(automaton.Alt(
			automaton.Lit(automaton.Span('a', 'b')),
			automaton.Lit(automaton.Span('0', '9')),
		))},
	})

	if err != nil {
		t.Fatal(err)
	}

	out := d.String()

	if strings.Contains(out, "nothing->") {
		t.Errorf("the table holds an edge that reads nothing:\n%s", out)
	}

	for s := automaton.State(1); int(s) <= d.States(); s++ {
		for c := automaton.Class(1); int(c) <= d.Classes().Len(); c++ {
			if at := d.Next(s, c); at != automaton.NoState && d.Classes().Part(c).Empty() {
				t.Errorf("state %d leaves on an empty class to %d", s, at)
			}
		}
	}

	t.Logf("star of an alternation, as a table:\n%s", out)
}
