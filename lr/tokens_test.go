package lr

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/cptaffe/automaton/dot"
	"github.com/cptaffe/automaton/fa"
)

func toShape(rs []Rule) []dot.Rule {
	out := make([]dot.Rule, len(rs))

	for i, r := range rs {
		out[i] = dot.Rule{Name: r.Name, Alts: r.Alts, Start: i == 0}
	}

	return out
}

func grammar(rs ...[2]string) []Rule {
	out := make([]Rule, 0, len(rs))

	for i, r := range rs {
		out = append(out, Rule{Name: r[0], Alts: strings.Split(r[1], "|"), At: i})
	}

	return out
}

func rows(d *fa.DFA[int], words []string) []string {
	out := make([]string, 0, d.States()+2)
	out = append(out, fmt.Sprintf("start %d", d.Start()))

	for s := fa.State(0); int(s) <= d.States(); s++ {
		row := fmt.Sprint(s)

		if d.Accept(s) != fa.NoMatch {
			row += " accept"
		}

		for i, w := range words {
			if to := d.Step(s, i+1); to != fa.NoState {
				row += fmt.Sprintf(" %s->%d", w, to)
			}
		}

		out = append(out, row)
	}

	return out
}

func reads(t *testing.T, m *Tokens, words ...string) bool {
	t.Helper()

	s := m.DFA().Start()

	for _, w := range words {
		if _, ok := m.at[w]; !ok {
			t.Fatalf("%s is not a token of this grammar", w)
		}

		s = m.Step(s, w)
	}

	return m.DFA().Accept(s) != fa.NoMatch
}

func TestRightRecursionIsExact(t *testing.T) {
	rs := grammar(
		[2]string{"list", "ITEM| ITEM COMMA list"},
	)

	if nested := dot.Nested(toShape(rs)); len(nested) > 0 {
		t.Fatalf("right recursion should not self-embed, got %v", nested)
	}

	m := tokens(t, rs)

	for _, want := range [][]string{
		{"ITEM"},
		{"ITEM", "COMMA", "ITEM"},
		{"ITEM", "COMMA", "ITEM", "COMMA", "ITEM"},
	} {
		if !reads(t, m, want...) {
			t.Errorf("%v is a list and is not accepted", want)
		}
	}

	if reads(t, m, "ITEM", "COMMA") {
		t.Error("a trailing comma is not a list")
	}
}

func TestSelfEmbeddingReadsMore(t *testing.T) {
	rs := grammar(
		[2]string{"expr", "NUM| LP expr RP"},
	)

	if nested := dot.Nested(toShape(rs)); len(nested) != 1 || nested[0] != "expr" {
		t.Fatalf("expr should self-embed, got %v", nested)
	}

	m := tokens(t, rs)

	if !reads(t, m, "LP", "LP", "NUM", "RP", "RP") {
		t.Error("a doubly bracketed number is an expr and is not accepted")
	}

	if !reads(t, m, "LP", "LP", "NUM", "RP") {
		t.Error("the approximation should accept unbalanced brackets")
	}
}

func TestTokensIsTheMachineOverTheGrammarsWords(t *testing.T) {
	for _, c := range []struct {
		name  string
		given []Rule
		then  []string
	}{
		{
			"two words either way round",
			grammar([2]string{"pair", "A B| B A"}),
			[]string{"start 1", "0", "1 A->2 B->3", "2 B->4", "3 A->4", "4 accept"},
		},
		{
			"a list by right recursion",
			grammar([2]string{"list", "ITEM| ITEM COMMA list"}),
			[]string{"start 1", "0", "1 ITEM->2", "2 accept COMMA->1"},
		},
		{
			"lines by left recursion from nothing",
			grammar([2]string{"file", "lines"}, [2]string{"lines", "nothing| lines NEWLINE"}),
			[]string{"start 1", "0", "1 accept NEWLINE->1"},
		},
		{
			"brackets that self-embed",
			grammar([2]string{"expr", "NUM| LP expr RP"}),
			[]string{"start 1", "0", "1 NUM->2 LP->1", "2 accept RP->2"},
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			m := tokens(t, c.given)

			if got := rows(m.DFA(), m.Words()); !reflect.DeepEqual(got, c.then) {
				t.Errorf("the machine is\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(c.then, "\n"))
			}
		})
	}
}

func TestEachClassHoldsTheWordItReads(t *testing.T) {
	m := tokens(t, grammar([2]string{"pair", "A B| B A"}))

	got := make([][]string, 0, m.DFA().Partition().Len())

	for c := range m.DFA().Partition().All() {
		got = append(got, m.In(c))
	}

	if then := [][]string{{"A"}, {"B"}}; !reflect.DeepEqual(got, then) {
		t.Errorf("the classes hold %v; want %v", got, then)
	}
}

func TestAWordTheGrammarNeverNamesGoesNowhere(t *testing.T) {
	m := tokens(t, grammar([2]string{"pair", "A B| B A"}))

	got := m.Step(m.DFA().Start(), "C")

	if got != fa.NoState {
		t.Errorf("C moves the machine to %d", got)
	}
}

func TestAGrammarOfNoWordsHasNoMachine(t *testing.T) {
	m := tokens(t, grammar([2]string{"empty", "nothing"}))

	if m != nil {
		t.Errorf("the machine has %d states", m.DFA().States())
	}
}

func TestReadOrdersRulesByTheirFirstNumber(t *testing.T) {
	given := "\tb: B (3)\n\t$accept: b $end (0)\n\ta: A (2)\n\tb: C (1)\n\tb: B (3)\n"

	g, err := Read(strings.NewReader(given))

	if err != nil {
		t.Fatal(err)
	}

	then := []Rule{{Name: "b", Alts: []string{"B", "C"}, At: 1}, {Name: "a", Alts: []string{"A"}, At: 2}}

	if !reflect.DeepEqual(g.Rules, then) {
		t.Errorf("the rules are %+v; want %+v", g.Rules, then)
	}
}

func TestReadRefusesWhatIsNotAReport(t *testing.T) {
	for _, c := range []struct {
		name  string
		given string
		then  string
	}{
		{"a rule number too large for an int", "\tfile: lines (99999999999999999999)\n", "rule 99999999999999999999: "},
		{"nothing at all", "", "no grammar rules"},
		{"text with no rules in it", "state 0\n\n\t$accept: . file $end\n", "no grammar rules"},
	} {
		t.Run(c.name, func(t *testing.T) {
			_, err := Read(strings.NewReader(c.given))

			if err == nil || !strings.Contains(err.Error(), c.then) {
				t.Errorf("got %v; want an error saying %q", err, c.then)
			}
		})
	}
}

func tokens(t *testing.T, rs []Rule) *Tokens {
	t.Helper()
	m, err := tokensOf(rs)
	if err != nil {
		t.Fatal(err)
	}
	return m
}
