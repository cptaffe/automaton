package lr

import (
	"strings"
	"testing"

	"github.com/cptaffe/automaton/dot"
)

func toShape(rs []Rule) []dot.Rule {
	out := make([]dot.Rule, len(rs))

	for i, r := range rs {
		out[i] = dot.Rule{Name: r.Name, Alts: r.Alts, Start: i == 0}
	}

	return out
}

func grammar(rs ...[2]string) []Rule {
	var out []Rule

	for i, r := range rs {
		out = append(out, Rule{Name: r[0], Alts: strings.Split(r[1], "|"), At: i})
	}

	return out
}

func reads(t *testing.T, m *Tokens, words ...string) bool {
	t.Helper()

	at := map[string]rune{}

	for r, name := range m.of {
		at[name] = r
	}

	s := m.dfa.Start()

	for _, w := range words {
		r, ok := at[w]

		if !ok {
			t.Fatalf("%s is not a token of this grammar", w)
		}

		if s = m.dfa.Step(s, r); s < 0 {
			return false
		}
	}

	_, ok := m.Accepts(s)

	return ok
}

func TestAnEmptyRuleAcceptsAtOnce(t *testing.T) {
	m := tokensOf(grammar(
		[2]string{"file", "lines"},
		[2]string{"lines", "nothing| lines NEWLINE"},
	), "file")

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

	m := tokensOf(rs, "list")

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

	m := tokensOf(rs, "expr")

	if !reads(t, m, "LP", "LP", "NUM", "RP", "RP") {
		t.Error("a doubly bracketed number is an expr and is not accepted")
	}

	if !reads(t, m, "LP", "LP", "NUM", "RP") {
		t.Error("the approximation should accept unbalanced brackets")
	}
}
