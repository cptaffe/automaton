package draw

import (
	"testing"

	"github.com/cptaffe/automaton/fa"
	"github.com/cptaffe/automaton/lex/lextest"
)

func TestANoteIsTheSummaryUnderTheScopesName(t *testing.T) {
	_, ms := lextest.Build(t, "start {\n  \"ab\"  emit x\n}\n")

	got := machine{m: ms[0]}.Note()

	if want := "start -- 1 rule, 3 states, 2 classes"; got != want {
		t.Errorf("the note is %q; want %q", got, want)
	}
}

func TestAnEmptySetIsLabelledNothing(t *testing.T) {
	given := fa.Set[rune]{}

	got := label(given)

	if got != "nothing" {
		t.Errorf("the label is %q; want nothing", got)
	}
}
