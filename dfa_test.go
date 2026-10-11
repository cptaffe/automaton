package automaton_test

import (
	"strings"
	"testing"
	"unicode"

	"github.com/cptaffe/automaton"
)

func text(s string) automaton.Expr {
	var e []automaton.Expr

	for _, r := range s {
		e = append(e, automaton.Lit(automaton.One(r)))
	}

	return automaton.Cat(e...)
}

func run(d *automaton.DFA, in string) (automaton.Match, int) {
	at, rule, upto := d.Start(), automaton.NoMatch, 0

	if a := d.Accept(at); a != automaton.NoMatch {
		rule = a
	}

	for i, r := range in {
		to := d.Step(at, r)

		if to == automaton.NoState {
			break
		}

		at = to

		if a := d.Accept(at); a != automaton.NoMatch {
			rule, upto = a, i+len(string(r))
		}
	}

	return rule, upto
}

func TestLongestMatchThenRuleOrder(t *testing.T) {
	rules := []automaton.Rule{
		{Name: "do", Expr: text("do")},
		{Name: "name", Expr: automaton.Plus(automaton.Lit(automaton.Span('a', 'z')))},
	}

	d, err := automaton.Build(rules)

	if err != nil {
		t.Fatal(err)
	}

	for _, c := range []struct {
		in   string
		rule automaton.Match
		upto int
	}{
		{"do", 1, 2},
		{"double", 2, 6},
		{"d", 2, 1},
		{"doing", 2, 5},
	} {
		rule, upto := run(d, c.in)

		if rule != c.rule || upto != c.upto {
			t.Errorf("%q matches rule %d upto %d; want %d upto %d", c.in, rule, upto, c.rule, c.upto)
		}
	}
}

func TestItReadsPastTheMatchAndReturns(t *testing.T) {
	d, err := automaton.Build([]automaton.Rule{
		{Name: "arrow", Expr: text("->")},
		{Name: "dash", Expr: text("-")},
	})

	if err != nil {
		t.Fatal(err)
	}

	if rule, upto := run(d, "->"); rule != 1 || upto != 2 {
		t.Errorf(`"->" is rule %d upto %d; want arrow upto 2`, rule, upto)
	}

	if rule, upto := run(d, "-x"); rule != 2 || upto != 1 {
		t.Errorf(`"-x" is rule %d upto %d; want dash upto 1, having read the x`, rule, upto)
	}
}

func TestTheOverrunIsUnbounded(t *testing.T) {
	digits := automaton.Plus(automaton.Lit(automaton.Span('0', '9')))
	d, err := automaton.Build([]automaton.Rule{
		{Name: "float", Expr: automaton.Cat(digits, text("."), digits)},
		{Name: "int", Expr: digits},
	})

	if err != nil {
		t.Fatal(err)
	}

	if rule, upto := run(d, "12.5"); rule != 1 || upto != 4 {
		t.Errorf(`"12.5" is rule %d upto %d; want float upto 4`, rule, upto)
	}

	if rule, upto := run(d, "12.foo"); rule != 2 || upto != 2 {
		t.Errorf(`"12.foo" is rule %d upto %d; want int upto 2`, rule, upto)
	}
}

func TestARuleCoveredEverywhereIsReported(t *testing.T) {
	rules := []automaton.Rule{
		{Name: "name", Expr: automaton.Plus(automaton.Lit(automaton.Span('a', 'z')))},
		{Name: "do", Expr: text("do")},
	}

	d, err := automaton.Build(rules)

	if err != nil {
		t.Fatal(err)
	}

	if got := d.Unreachable(rules); len(got) != 1 || got[0] != "do" {
		t.Errorf("the dead rules are %v; `name` is written first and covers `do`", got)
	}
}

func TestALiveRuleIsNotReported(t *testing.T) {
	rules := []automaton.Rule{
		{Name: "do", Expr: text("do")},
		{Name: "name", Expr: automaton.Plus(automaton.Lit(automaton.Span('a', 'z')))},
	}

	d, err := automaton.Build(rules)

	if err != nil {
		t.Fatal(err)
	}

	if got := d.Unreachable(rules); len(got) != 0 {
		t.Errorf("the dead rules are %v; both are reachable", got)
	}
}

func TestARuleThatMatchesNothingIsRefused(t *testing.T) {
	_, err := automaton.Build([]automaton.Rule{{Name: "nothing", Expr: automaton.Empty()}})

	if err == nil || !strings.Contains(err.Error(), "never advance") {
		t.Errorf("the refusal is %v", err)
	}
}

func TestStarDoesNotDuplicateItsBody(t *testing.T) {
	d, err := automaton.Build([]automaton.Rule{
		{Name: "word", Expr: automaton.Plus(automaton.Lit(automaton.Span('a', 'z')))},
	})

	if err != nil {
		t.Fatal(err)
	}

	if d.States() != 2 {
		t.Errorf("[a-z]+ makes %d states; want 2:\n%s", d.States(), d)
	}
}

func TestMinimisingMergesStatesNothingTellsApart(t *testing.T) {
	rules := []automaton.Rule{
		{Name: "word", Expr: automaton.Alt(text("ab"), text("cb"))},
	}

	d, err := automaton.Build(rules)

	if err != nil {
		t.Fatal(err)
	}

	if d.States() != 5 {
		t.Fatalf("the reachable machine has %d states; want 5:\n%s", d.States(), d)
	}

	m := d.Minimise()

	if m.States() != 3 {
		t.Errorf("the minimal machine has %d states; want 3:\n%s", m.States(), m)
	}

	for _, c := range []struct {
		in   string
		want automaton.Match
	}{{"ab", 1}, {"cb", 1}, {"ax", automaton.NoMatch}, {"b", automaton.NoMatch}} {
		rule, _ := run(m, c.in)

		if rule != c.want {
			t.Errorf("minimised, %q is rule %d; want %d", c.in, rule, c.want)
		}
	}
}

func TestMinimisingKeepsRulesApart(t *testing.T) {
	rules := []automaton.Rule{
		{Name: "ab", Expr: text("ab")},
		{Name: "ac", Expr: text("ac")},
	}

	d, err := automaton.Build(rules)

	if err != nil {
		t.Fatal(err)
	}

	m := d.Minimise()

	if got := m.States(); got != d.States() {
		t.Errorf("minimising took %d states to %d; the two accepts are distinct", d.States(), got)
	}

	for _, c := range []struct {
		in   string
		want automaton.Match
	}{{"ab", 1}, {"ac", 2}} {
		if rule, _ := run(m, c.in); rule != c.want {
			t.Errorf("minimised, %q is rule %d; want %d", c.in, rule, c.want)
		}
	}
}

func TestAUnicodeRuleStaysSmall(t *testing.T) {
	letter := automaton.Lit(automaton.Category(unicode.L))
	digit := automaton.Lit(automaton.Category(unicode.Nd))

	d, err := automaton.Build([]automaton.Rule{
		{Name: "name", Expr: automaton.Cat(letter, automaton.Star(automaton.Alt(letter, digit)))},
		{Name: "number", Expr: automaton.Plus(digit)},
	})

	if err != nil {
		t.Fatal(err)
	}

	if d.Classes().Len() != 2 {
		t.Errorf("the alphabet is %d classes; letters and digits are two", d.Classes().Len())
	}

	if d.States() > 6 {
		t.Errorf("the machine has %d states for two rules over all of unicode:\n%s", d.States(), d)
	}

	if at := d.Step(d.Start(), '漢'); at == automaton.NoState || d.Accept(at) != 1 {
		t.Error("a han character does not begin a name")
	}

	if at := d.Step(d.Start(), '٣'); at == automaton.NoState || d.Accept(at) != 2 {
		t.Error("an arabic-indic three is not a number")
	}
}
