package dot_test

import (
	"strings"
	"testing"

	"github.com/cptaffe/automaton"
	"github.com/cptaffe/automaton/dot"
)

type toy struct {
	c       *automaton.Classes
	letters automaton.Class
}

func (t toy) Name() string { return "toy" }
func (t toy) Note() string { return "" }
func (t toy) States() int  { return 3 }
func (t toy) Starts() map[string]automaton.State {
	return map[string]automaton.State{"start": 0}
}
func (t toy) Classes() *automaton.Classes { return t.c }

func (t toy) Next(s automaton.State, c automaton.Class) automaton.State {
	if s != 0 {
		return automaton.NoState
	}

	if c == t.letters {
		return 1
	}

	return 2
}

func (t toy) Accepts(s automaton.State) (string, bool) {
	switch s {
	case 1:
		return "word", true
	case 2:
		return "number", true
	}

	return "", false
}

func TestAnEdgeIsLabelledWithItsRange(t *testing.T) {
	letters, digits := automaton.Span('a', 'z'), automaton.Span('0', '9')
	c := automaton.Partition([]automaton.Set{letters, digits})
	out := string(dot.Draw(toy{c: c, letters: c.Covers(letters)[0]}))

	if n := strings.Count(out, "->"); n != 2 {
		t.Errorf("the drawing has %d edges; two classes go two places:\n%s", n, out)
	}

	for _, want := range []string{`" a-z "`, `" 0-9 "`, "doublecircle", "word", "number", `label="start"`} {
		if !strings.Contains(out, want) {
			t.Errorf("the drawing does not hold %s:\n%s", want, out)
		}
	}
}

func TestClassesToOneStateAreOneEdge(t *testing.T) {
	c := automaton.Partition([]automaton.Set{automaton.Span('a', 'z'), automaton.Span('0', '9')})
	out := string(dot.Draw(toy{c: c, letters: -1}))

	if n := strings.Count(out, "->"); n != 1 {
		t.Errorf("the drawing has %d edges; both classes go to one state:\n%s", n, out)
	}

	if !strings.Contains(out, `" 0-9a-z "`) {
		t.Errorf("the one edge is not labelled with both ranges:\n%s", out)
	}
}
