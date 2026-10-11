package draw_test

import (
	"strings"
	"testing"

	"github.com/cptaffe/automaton/dot"
	"github.com/cptaffe/automaton/lr"
	"github.com/cptaffe/automaton/lr/draw"
)

func TestTheDrawingsSayWhatTheGrammarIs(t *testing.T) {
	for _, c := range []struct {
		rules []lr.Rule
		note  string
		edge  string
	}{
		{[]lr.Rule{{Name: "list", Alts: []string{"ITEM", "ITEM COMMA list"}}}, "exactly the grammar", "COMMA"},
		{[]lr.Rule{{Name: "expr", Alts: []string{"NUM", "LP expr RP"}}}, "expr self-embeds", "LP"},
		{[]lr.Rule{{Name: "word", Alts: []string{"A", "B", "C", "D", "E", "F"}}}, "exactly the grammar", "and 2 more"},
	} {
		g := &lr.Grammar{Rules: c.rules}
		m := draw.Tokens(g, "g")
		b, err := dot.Draw(m)
		if err != nil {
			t.Fatal(err)
		}
		if out := string(b); !strings.Contains(m.Note(), c.note) || !strings.Contains(out, c.edge) {
			t.Errorf("%v: note %q, drawing:\n%s", c.rules, m.Note(), out)
		}
		if _, err := draw.Shape(g, "g"); err != nil {
			t.Error(err)
		}
	}
	if draw.Tokens(&lr.Grammar{Rules: []lr.Rule{{Name: "a", Alts: []string{"nothing"}}}}, "g") != nil {
		t.Error("a grammar of no tokens has a machine over tokens")
	}
}
