package fa_test

import (
	"strings"
	"testing"
	"unicode"

	"github.com/cptaffe/automaton/fa"
)

func text(s string) fa.Expr[rune] {
	e := make([]fa.Expr[rune], 0, len(s))

	for _, r := range s {
		e = append(e, fa.Lit(fa.One(r)))
	}

	return fa.Cat(e...)
}

func run(d *fa.DFA[rune], in string) (fa.Match, int) {
	at, rule, upto := d.Start(), fa.NoMatch, 0

	if a := d.Accept(at); a != fa.NoMatch {
		rule = a
	}

	for i, r := range in {
		to := d.Step(at, r)

		if to == fa.NoState {
			break
		}

		at = to

		if a := d.Accept(at); a != fa.NoMatch {
			rule, upto = a, i+len(string(r))
		}
	}

	return rule, upto
}

func TestLongestMatchThenRuleOrder(t *testing.T) {
	rules := []fa.Rule[rune]{
		{Name: "do", Expr: text("do")},
		{Name: "name", Expr: fa.Plus(fa.Lit(fa.Span('a', 'z')))},
	}

	d, err := fa.Build(rules)

	if err != nil {
		t.Fatal(err)
	}

	for _, c := range []struct {
		in   string
		rule fa.Match
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
	d, err := fa.Build([]fa.Rule[rune]{
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
	digits := fa.Plus(fa.Lit(fa.Span('0', '9')))
	d, err := fa.Build([]fa.Rule[rune]{
		{Name: "float", Expr: fa.Cat(digits, text("."), digits)},
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
	rules := []fa.Rule[rune]{
		{Name: "name", Expr: fa.Plus(fa.Lit(fa.Span('a', 'z')))},
		{Name: "do", Expr: text("do")},
	}

	d, err := fa.Build(rules)

	if err != nil {
		t.Fatal(err)
	}

	if got := d.Unreachable(rules); len(got) != 1 || got[0] != "do" {
		t.Errorf("the dead rules are %v; `name` is written first and covers `do`", got)
	}
}

func TestALiveRuleIsNotReported(t *testing.T) {
	rules := []fa.Rule[rune]{
		{Name: "do", Expr: text("do")},
		{Name: "name", Expr: fa.Plus(fa.Lit(fa.Span('a', 'z')))},
	}

	d, err := fa.Build(rules)

	if err != nil {
		t.Fatal(err)
	}

	if got := d.Unreachable(rules); len(got) != 0 {
		t.Errorf("the dead rules are %v; both are reachable", got)
	}
}

func TestARuleThatMatchesNothingIsRefused(t *testing.T) {
	_, err := fa.Build([]fa.Rule[rune]{{Name: "nothing", Expr: fa.Empty[rune]()}})

	if err == nil || !strings.Contains(err.Error(), "never advance") {
		t.Errorf("the refusal is %v", err)
	}
}

func TestStarDoesNotDuplicateItsBody(t *testing.T) {
	d, err := fa.Build([]fa.Rule[rune]{
		{Name: "word", Expr: fa.Plus(fa.Lit(fa.Span('a', 'z')))},
	})

	if err != nil {
		t.Fatal(err)
	}

	if d.States() != 2 {
		t.Errorf("[a-z]+ makes %d states; want 2:\n%s", d.States(), d)
	}
}

func TestMinimisingMergesStatesNothingTellsApart(t *testing.T) {
	rules := []fa.Rule[rune]{
		{Name: "word", Expr: fa.Alt(text("ab"), text("cb"))},
	}

	d, err := fa.Build(rules)

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
		want fa.Match
	}{{"ab", 1}, {"cb", 1}, {"ax", fa.NoMatch}, {"b", fa.NoMatch}} {
		rule, _ := run(m, c.in)

		if rule != c.want {
			t.Errorf("minimised, %q is rule %d; want %d", c.in, rule, c.want)
		}
	}
}

func TestMinimisingKeepsRulesApart(t *testing.T) {
	rules := []fa.Rule[rune]{
		{Name: "ab", Expr: text("ab")},
		{Name: "ac", Expr: text("ac")},
	}

	d, err := fa.Build(rules)

	if err != nil {
		t.Fatal(err)
	}

	m := d.Minimise()

	if got := m.States(); got != d.States() {
		t.Errorf("minimising took %d states to %d; the two accepts are distinct", d.States(), got)
	}

	for _, c := range []struct {
		in   string
		want fa.Match
	}{{"ab", 1}, {"ac", 2}} {
		if rule, _ := run(m, c.in); rule != c.want {
			t.Errorf("minimised, %q is rule %d; want %d", c.in, rule, c.want)
		}
	}
}

func TestAUnicodeRuleStaysSmall(t *testing.T) {
	letter := fa.Lit(fa.Category(unicode.L))
	digit := fa.Lit(fa.Category(unicode.Nd))

	d, err := fa.Build([]fa.Rule[rune]{
		{Name: "name", Expr: fa.Cat(letter, fa.Star(fa.Alt(letter, digit)))},
		{Name: "number", Expr: fa.Plus(digit)},
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

	if at := d.Step(d.Start(), '漢'); at == fa.NoState || d.Accept(at) != 1 {
		t.Error("a han character does not begin a name")
	}

	if at := d.Step(d.Start(), '٣'); at == fa.NoState || d.Accept(at) != 2 {
		t.Error("an arabic-indic three is not a number")
	}
}

func TestTheTableHasNoEmptyTransitions(t *testing.T) {
	d, err := fa.Build([]fa.Rule[rune]{
		{Name: "word", Expr: fa.Star(fa.Alt(
			fa.Lit(fa.Span('a', 'b')),
			fa.Lit(fa.Span('0', '9')),
		))},
	})

	if err != nil {
		t.Fatal(err)
	}

	out := d.String()

	if strings.Contains(out, "nothing->") {
		t.Errorf("the table holds an edge that reads nothing:\n%s", out)
	}

	for s := fa.State(1); int(s) <= d.States(); s++ {
		for c := fa.Class(1); int(c) <= d.Classes().Len(); c++ {
			if at := d.Next(s, c); at != fa.NoState && d.Classes().Part(c).Empty() {
				t.Errorf("state %d leaves on an empty class to %d", s, at)
			}
		}
	}

	t.Logf("star of an alternation, as a table:\n%s", out)
}

func TestAStateKnowsItsRules(t *testing.T) {
	one := func(x uint16) fa.Expr[uint16] { return fa.Lit(fa.One(x)) }
	d, err := fa.Build([]fa.Rule[uint16]{
		{Name: "a", Expr: one(1)},
		{Name: "ab", Expr: fa.Cat(one(1), one(2))},
		{Name: "a+", Expr: fa.Plus(one(1))},
	})
	if err != nil {
		t.Fatal(err)
	}
	at := d.Step(d.Start(), 1)
	if got := d.Done(at); len(got) != 2 || got[0] != 1 || got[1] != 3 {
		t.Errorf("after 1, done is %v; want a and a+", got)
	}
	if got := d.Going(at); len(got) != 2 || got[0] != 2 || got[1] != 3 {
		t.Errorf("after 1, going is %v; want ab and a+", got)
	}
	if d.Accept(at) != 1 {
		t.Errorf("after 1, the winner is %d; want a", d.Accept(at))
	}
}

func TestAFallbackTakesWhatNothingElseDoes(t *testing.T) {
	d, err := fa.Build([]fa.Rule[rune]{
		{Expr: fa.Cat(fa.Lit(fa.One('x')), fa.Lit(fa.One('s'))), Name: "save"},
		{Expr: fa.Cat(fa.Lit(fa.One('x')), fa.Fallback(fa.Span('a', 'z'))), Name: "other"},
	})
	if err != nil {
		t.Fatal(err)
	}
	at := d.Step(d.Start(), 'x')
	if got := d.Accept(d.Step(at, 's')); got != 1 {
		t.Errorf("x s finishes %d; want the specific rule, 1", got)
	}
	if got := d.Accept(d.Step(at, 'q')); got != 2 {
		t.Errorf("x q finishes %d; want the fallback, 2", got)
	}
}
