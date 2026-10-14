package fa_test

import (
	"testing"

	"github.com/cptaffe/automaton/fa"
)

func machine(t *testing.T, es ...fa.Expr[rune]) *fa.DFA[rune] {
	t.Helper()

	rules := make([]fa.Rule[rune], 0, len(es))

	for _, e := range es {
		rules = append(rules, fa.Rule[rune]{Expr: e})
	}

	d, err := fa.Build(rules)

	if err != nil {
		t.Fatal(err)
	}

	return d
}

func lit(s string) fa.Expr[rune] { return fa.Lit(fa.Text(s)) }

func TestDistinguishFindsAnInputTwoMachinesDisagreeOn(t *testing.T) {
	a, b, c := lit("a"), lit("b"), lit("c")

	for _, c := range []struct {
		name  string
		given [2][]fa.Expr[rune]
		then  string
	}{
		{"one language built two ways", [2][]fa.Expr[rune]{
			{fa.Cat(a, fa.Alt(b, c))},
			{fa.Alt(fa.Cat(a, b), fa.Cat(a, c))},
		}, ""},
		{"a repeat and its unrolling", [2][]fa.Expr[rune]{
			{fa.Plus(a)},
			{fa.Cat(a, fa.Star(a))},
		}, ""},
		{"a longer word", [2][]fa.Expr[rune]{
			{fa.Cat(a, b)},
			{fa.Cat(a, b, c)},
		}, "ab"},
		{"a narrower class", [2][]fa.Expr[rune]{
			{fa.Cat(fa.Lit(fa.Span('a', 'm')), lit("x"))},
			{fa.Cat(fa.Lit(fa.Span('a', 'z')), lit("x"))},
		}, "nx"},
		{"another rule wins", [2][]fa.Expr[rune]{
			{a, b},
			{b, a},
		}, "a"},
	} {
		t.Run(c.name, func(t *testing.T) {
			x, y := machine(t, c.given[0]...), machine(t, c.given[1]...)

			got, differ := fa.Distinguish(x, y)

			if string(got) != c.then || differ != (c.then != "") || fa.Equal(x, y) == differ {
				t.Errorf("got %q, %v; want %q", string(got), differ, c.then)
			}
		})
	}
}

func TestAMinimisedMachineEqualsItsOriginal(t *testing.T) {
	given := machine(t, fa.Cat(fa.Star(fa.Alt(lit("a"), lit("b"))), lit("abb")), lit("ba"))

	when := given.Minimise()

	if !fa.Equal(given, when) || when.States() >= given.States() {
		t.Errorf("minimising %d states to %d changed the language", given.States(), when.States())
	}
}
