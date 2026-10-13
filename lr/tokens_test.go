package lr

import (
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

func TestAnEmptyRuleAcceptsAtOnce(t *testing.T) {
	m := tokensOf(grammar(
		[2]string{"file", "lines"},
		[2]string{"lines", "nothing| lines NEWLINE"},
	))

	if m == nil {
		t.Fatal("no machine")
	}

	if !reads(t, m) {
		t.Error("a file of no lines is a file, but the machine does not accept it")
	}

	if !reads(t, m, "NEWLINE", "NEWLINE") {
		t.Error("two lines are not accepted")
	}
}

func TestRightRecursionIsExact(t *testing.T) {
	rs := grammar(
		[2]string{"list", "ITEM| ITEM COMMA list"},
	)

	if nested := dot.Nested(toShape(rs)); len(nested) > 0 {
		t.Fatalf("right recursion should not self-embed, got %v", nested)
	}

	m := tokensOf(rs)

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

	m := tokensOf(rs)

	if !reads(t, m, "LP", "LP", "NUM", "RP", "RP") {
		t.Error("a doubly bracketed number is an expr and is not accepted")
	}

	if !reads(t, m, "LP", "LP", "NUM", "RP") {
		t.Error("the approximation should accept unbalanced brackets")
	}
}

func TestAClassNamesItsTokens(t *testing.T) {
	m := tokensOf(grammar([2]string{"pair", "A B| B A"}))

	if got := m.Words(); len(got) != 2 {
		t.Fatalf("the words are %v; want A and B", got)
	}

	var seen []string

	for c := fa.Class(1); int(c) <= m.DFA().Classes().Len(); c++ {
		seen = append(seen, m.In(c)...)
	}

	if strings.Join(seen, " ") != "A B" && strings.Join(seen, " ") != "B A" {
		t.Errorf("the classes hold %v; want A and B once each", seen)
	}

	if m.Step(m.DFA().Start(), "C") != fa.NoState {
		t.Error("a word the grammar never names moves the machine")
	}
}

func TestReadRefusesABadRuleNumber(t *testing.T) {
	if _, err := Read(strings.NewReader("\tfile: lines (99999999999999999999)\n")); err == nil {
		t.Error("a rule number too large for an int is read without complaint")
	}
}
