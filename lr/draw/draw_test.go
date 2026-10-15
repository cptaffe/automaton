package draw_test

import (
	"reflect"
	"testing"

	"github.com/cptaffe/automaton/dot"
	"github.com/cptaffe/automaton/internal/dottest"
	"github.com/cptaffe/automaton/lr"
	"github.com/cptaffe/automaton/lr/draw"
)

type drawing struct {
	note  string
	nodes []string
	edges []string
}

func TestTokensDrawsTheMachineOverTheGrammar(t *testing.T) {
	for _, c := range []struct {
		name  string
		given []lr.Rule
		then  drawing
	}{
		{
			"a list that nothing embeds",
			[]lr.Rule{{Name: "list", Alts: []string{"ITEM", "ITEM COMMA list"}}},
			drawing{
				"g over tokens -- 2 states, 2 tokens; nothing self-embeds, so this is exactly the grammar",
				[]string{"g (start)", "2 (accept)"},
				[]string{"g -ITEM-> 2", "2 -COMMA-> g"},
			},
		},
		{
			"brackets that self-embed",
			[]lr.Rule{{Name: "expr", Alts: []string{"NUM", "LP expr RP"}}},
			drawing{
				"g over tokens -- 2 states, 3 tokens; expr self-embeds, so this reads more than the grammar does",
				[]string{"g (start)", "2 (accept)"},
				[]string{"g -LP-> g", "g -NUM-> 2", "2 -RP-> 2"},
			},
		},
		{
			"two rules that self-embed",
			[]lr.Rule{{Name: "a", Alts: []string{"L b R", "N"}}, {Name: "b", Alts: []string{"a"}}},
			drawing{
				"g over tokens -- 2 states, 3 tokens; a, b self-embed, so this reads more than the grammar does",
				[]string{"g (start)", "2 (accept)"},
				[]string{"g -L-> g", "g -N-> 2", "2 -R-> 2"},
			},
		},
		{
			"four words on one move",
			[]lr.Rule{{Name: "word", Alts: []string{"D", "C", "B", "A"}}},
			drawing{
				"g over tokens -- 2 states, 4 tokens; nothing self-embeds, so this is exactly the grammar",
				[]string{"g (start)", "2 (accept)"},
				[]string{"g -A B C D-> 2"},
			},
		},
		{
			"six words on one move",
			[]lr.Rule{{Name: "word", Alts: []string{"A", "B", "C", "D", "E", "F"}}},
			drawing{
				"g over tokens -- 2 states, 6 tokens; nothing self-embeds, so this is exactly the grammar",
				[]string{"g (start)", "2 (accept)"},
				[]string{"g -A B C D and 2 more-> 2"},
			},
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			m := tokens(t, &lr.Grammar{Rules: c.given}, "g")
			g := dot.Draw(m)

			if got := (drawing{m.Note(), dottest.Nodes(g), dottest.Edges(g)}); !reflect.DeepEqual(got, c.then) {
				t.Errorf("the drawing is\n%q\nwant\n%q", got, c.then)
			}
		})
	}
}

func TestAGrammarOfNoTokensHasNoMachine(t *testing.T) {
	given := &lr.Grammar{Rules: []lr.Rule{{Name: "a", Alts: []string{"nothing"}}}}

	m := tokens(t, given, "g")

	if m != nil {
		t.Errorf("the machine has %d states", m.States())
	}
}

func TestShapeDrawsEachRuleAndWhatItNames(t *testing.T) {
	for _, c := range []struct {
		name  string
		given []lr.Rule
		then  drawing
	}{
		{
			"lines by left recursion",
			[]lr.Rule{{Name: "file", Alts: []string{"lines"}}, {Name: "lines", Alts: []string{"nothing", "lines NEWLINE"}}},
			drawing{
				"",
				[]string{"file lines (start)", "lines nothing lines NEWLINE"},
				[]string{"file lines --> lines nothing lines NEWLINE", "lines nothing lines NEWLINE --> lines nothing lines NEWLINE (cycle)"},
			},
		},
		{
			"brackets that self-embed",
			[]lr.Rule{{Name: "expr", Alts: []string{"NUM", "LP expr RP"}}},
			drawing{
				"",
				[]string{"expr NUM LP expr RP (start)"},
				[]string{"expr NUM LP expr RP --> expr NUM LP expr RP (cycle nest)"},
			},
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			g := draw.Shape(&lr.Grammar{Rules: c.given}, "g")

			if got := (drawing{g.Note, dottest.Nodes(g), dottest.Edges(g)}); !reflect.DeepEqual(got, c.then) {
				t.Errorf("the shape is\n%q\nwant\n%q", got, c.then)
			}
		})
	}
}

func tokens(t *testing.T, g *lr.Grammar, name string) dot.Machine {
	t.Helper()
	m, err := draw.Tokens(g, name)
	if err != nil {
		t.Fatal(err)
	}
	return m
}
