package fa_test

import (
	"fmt"
	"reflect"
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

func build[S fa.Symbol](t *testing.T, rules []fa.Rule[S]) *fa.DFA[S] {
	t.Helper()

	d, err := fa.Build(rules)

	if err != nil {
		t.Fatal(err)
	}

	return d
}

type read struct {
	rule fa.Match
	upto int
}

func run(d *fa.DFA[rune], in string) read {
	at, got := d.Start(), read{d.Accept(d.Start()), 0}

	for i, r := range in {
		to := d.Step(at, r)

		if to == fa.NoState {
			break
		}

		at = to

		if a := d.Accept(at); a != fa.NoMatch {
			got = read{a, i + len(string(r))}
		}
	}

	return got
}

func table[S fa.Symbol](d *fa.DFA[S]) []string {
	out := make([]string, 0, d.States()+2)
	out = append(out, fmt.Sprintf("start %d", d.Start()))

	for s := fa.State(0); int(s) <= d.States(); s++ {
		row := fmt.Sprintf("%d done %v going %v", s, d.Done(s), d.Going(s))

		for c, part := range d.Partition().All() {
			if to := d.Next(s, c); to != fa.NoState {
				row += fmt.Sprintf(" %s->%d", part, to)
			}
		}

		out = append(out, row)
	}

	return out
}

func TestAMachineTakesTheLongestMatchThenTheFirstRule(t *testing.T) {
	words := []fa.Rule[rune]{
		{Name: "do", Expr: text("do")},
		{Name: "name", Expr: fa.Plus(fa.Lit(fa.Span('a', 'z')))},
	}
	arrows := []fa.Rule[rune]{
		{Name: "arrow", Expr: text("->")},
		{Name: "dash", Expr: text("-")},
	}
	digits := fa.Plus(fa.Lit(fa.Span('0', '9')))
	numbers := []fa.Rule[rune]{
		{Name: "float", Expr: fa.Cat(digits, text("."), digits)},
		{Name: "int", Expr: digits},
	}

	for _, c := range []struct {
		name  string
		given []fa.Rule[rune]
		when  string
		then  read
	}{
		{"a keyword", words, "do", read{1, 2}},
		{"a name that begins with a keyword", words, "double", read{2, 6}},
		{"a prefix of a keyword", words, "d", read{2, 1}},
		{"a keyword and then more letters", words, "doing", read{2, 5}},
		{"nothing any rule reads", words, "9", read{fa.NoMatch, 0}},
		{"the longer of two rules", arrows, "->", read{1, 2}},
		{"the shorter after reading on", arrows, "-x", read{2, 1}},
		{"a float", numbers, "12.5", read{1, 4}},
		{"an int after reading past a dot", numbers, "12.foo", read{2, 2}},
	} {
		t.Run(c.name, func(t *testing.T) {
			if got := run(build(t, c.given), c.when); got != c.then {
				t.Errorf("%q reads as %+v; want %+v", c.when, got, c.then)
			}
		})
	}
}

func TestOptReadsItsBodyOrNothing(t *testing.T) {
	sign := []fa.Rule[rune]{{Name: "number", Expr: fa.Cat(fa.Opt(fa.Lit(fa.One('-'))), fa.Plus(fa.Lit(fa.Span('0', '9'))))}}
	prefix := []fa.Rule[rune]{{Name: "word", Expr: fa.Cat(fa.Opt(text("ab")), text("c"))}}

	for _, c := range []struct {
		name  string
		given []fa.Rule[rune]
		when  string
		then  read
	}{
		{"without the sign", sign, "5", read{1, 1}},
		{"with the sign", sign, "-5", read{1, 2}},
		{"the sign twice", sign, "--5", read{fa.NoMatch, 0}},
		{"without the prefix", prefix, "c", read{1, 1}},
		{"with the prefix", prefix, "abc", read{1, 3}},
		{"half the prefix", prefix, "ac", read{fa.NoMatch, 0}},
	} {
		t.Run(c.name, func(t *testing.T) {
			if got := run(build(t, c.given), c.when); got != c.then {
				t.Errorf("%q reads as %+v; want %+v", c.when, got, c.then)
			}
		})
	}
}

func TestAcceptedLeavesOutARuleWrittenAfterOneThatCoversIt(t *testing.T) {
	name := fa.Rule[rune]{Name: "name", Expr: fa.Plus(fa.Lit(fa.Span('a', 'z')))}
	do := fa.Rule[rune]{Name: "do", Expr: text("do")}

	for _, c := range []struct {
		name  string
		given []fa.Rule[rune]
		then  []fa.Match
	}{
		{"name first covers do", []fa.Rule[rune]{name, do}, []fa.Match{1}},
		{"do first keeps both", []fa.Rule[rune]{do, name}, []fa.Match{1, 2}},
	} {
		t.Run(c.name, func(t *testing.T) {
			got := build(t, c.given).Accepted()

			if !reflect.DeepEqual(got, c.then) {
				t.Errorf("the rules accepted are %v; want %v", got, c.then)
			}
		})
	}
}

func TestARuleThatMatchesNothingIsRefused(t *testing.T) {
	_, err := fa.Build([]fa.Rule[rune]{{Name: "nothing", Expr: fa.Empty[rune]()}})

	if err == nil || !strings.Contains(err.Error(), "never advance") {
		t.Errorf("the refusal is %v", err)
	}
}

func TestClassesThatSplitASetAreRefused(t *testing.T) {
	given := fa.Split([]fa.Set[rune]{fa.Span('a', 'm'), fa.Span('h', 'z')})

	_, err := fa.BuildWith([]fa.Rule[rune]{{Name: "middle", Expr: fa.Lit(fa.Span('d', 'p'))}}, given, 0)

	if err == nil || !strings.Contains(err.Error(), "the classes split d-p") {
		t.Errorf("the refusal is %v", err)
	}
}

func TestStarDoesNotDuplicateItsBody(t *testing.T) {
	d := build(t, []fa.Rule[rune]{
		{Name: "word", Expr: fa.Plus(fa.Lit(fa.Span('a', 'z')))},
	})

	got := table(d)

	then := []string{
		"start 1",
		"0 done [] going []",
		"1 done [] going [1] a-z->2",
		"2 done [1] going [1] a-z->2",
	}

	if !reflect.DeepEqual(got, then) {
		t.Errorf("[a-z]+ is\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(then, "\n"))
	}
}

func TestMinimisingMergesStatesNothingTellsApart(t *testing.T) {
	d := build(t, []fa.Rule[rune]{
		{Name: "word", Expr: fa.Alt(text("ab"), text("cb"))},
	})

	full, small := table(d), table(d.Minimise())

	if then := []string{
		"start 1",
		"0 done [] going []",
		"1 done [] going [1] a->2 c->4",
		"2 done [] going [1] b->3",
		"3 done [1] going []",
		"4 done [] going [1] b->5",
		"5 done [1] going []",
	}; !reflect.DeepEqual(full, then) {
		t.Errorf("ab|cb is\n%s\nwant\n%s", strings.Join(full, "\n"), strings.Join(then, "\n"))
	}

	if then := []string{
		"start 1",
		"0 done [] going []",
		"1 done [] going [1] a->2 c->2",
		"2 done [] going [1] b->3",
		"3 done [1] going []",
	}; !reflect.DeepEqual(small, then) {
		t.Errorf("ab|cb minimised is\n%s\nwant\n%s", strings.Join(small, "\n"), strings.Join(then, "\n"))
	}
}

func TestMinimisingKeepsRulesApart(t *testing.T) {
	d := build(t, []fa.Rule[rune]{
		{Name: "ab", Expr: text("ab")},
		{Name: "ac", Expr: text("ac")},
	})

	got := table(d.Minimise())

	then := []string{
		"start 1",
		"0 done [] going []",
		"1 done [] going [1 2] a->2",
		"2 done [] going [1 2] b->3 c->4",
		"3 done [1] going []",
		"4 done [2] going []",
	}

	if !reflect.DeepEqual(got, then) {
		t.Errorf("ab and ac minimised are\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(then, "\n"))
	}
}

func TestAUnicodeRuleStaysSmall(t *testing.T) {
	letter := fa.Lit(fa.Category(unicode.L))
	digit := fa.Lit(fa.Category(unicode.Nd))

	d := build(t, []fa.Rule[rune]{
		{Name: "name", Expr: fa.Cat(letter, fa.Star(fa.Alt(letter, digit)))},
		{Name: "number", Expr: fa.Plus(digit)},
	})

	if d.Partition().Len() != 2 {
		t.Errorf("the alphabet is %d classes; letters and digits are two", d.Partition().Len())
	}

	if d.States() != 4 {
		t.Errorf("the machine has %d states for two rules over all of unicode; want 4", d.States())
	}

	if at := d.Step(d.Start(), '漢'); at == fa.NoState || d.Accept(at) != 1 {
		t.Error("a han character does not begin a name")
	}

	if at := d.Step(d.Start(), '٣'); at == fa.NoState || d.Accept(at) != 2 {
		t.Error("an arabic-indic three is not a number")
	}
}

func TestStringListsEachStateWithItsAcceptAndMoves(t *testing.T) {
	d := build(t, []fa.Rule[rune]{
		{Name: "num", Expr: fa.Plus(fa.Lit(fa.Span('0', '9')))},
		{Name: "word", Expr: fa.Star(fa.Alt(
			fa.Lit(fa.Span('a', 'b')),
			fa.Lit(fa.Span('0', '9')),
		))},
	})

	got := d.String()

	if then := "1 accept 2 0-9->2 ab->3\n2 accept 1 0-9->2 ab->3\n3 accept 2 0-9->3 ab->3\n"; got != then {
		t.Errorf("the table is\n%s\nwant\n%s", got, then)
	}
}

func TestAStateKnowsWhichRulesAreDoneAndWhichGoOn(t *testing.T) {
	one := func(x uint16) fa.Expr[uint16] { return fa.Lit(fa.One(x)) }
	d := build(t, []fa.Rule[uint16]{
		{Name: "a", Expr: one(1)},
		{Name: "ab", Expr: fa.Cat(one(1), one(2))},
		{Name: "a+", Expr: fa.Plus(one(1))},
	})

	got := table(d)

	then := []string{
		"start 1",
		"0 done [] going []",
		"1 done [] going [1 2 3] 1->2",
		"2 done [1 3] going [2 3] 1->3 2->4",
		"3 done [3] going [3] 1->3",
		"4 done [2] going []",
	}

	if !reflect.DeepEqual(got, then) {
		t.Errorf("a, ab and a+ are\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(then, "\n"))
	}
}

func TestAFallbackTakesWhatNothingElseDoes(t *testing.T) {
	d := build(t, []fa.Rule[rune]{
		{Name: "save", Expr: fa.Cat(fa.Lit(fa.One('x')), fa.Lit(fa.One('s')))},
		{Name: "other", Expr: fa.Cat(fa.Lit(fa.One('x')), fa.Fallback(fa.Span('a', 'z')))},
	})

	got := table(d)

	then := []string{
		"start 1",
		"0 done [] going []",
		"1 done [] going [1 2] x->2",
		"2 done [] going [1 2] a-rt-wyz->3 s->4 x->3",
		"3 done [2] going []",
		"4 done [1] going []",
	}

	if !reflect.DeepEqual(got, then) {
		t.Errorf("x s beside x and any letter is\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(then, "\n"))
	}
}

func TestABuildStopsAtItsBudgetOfStates(t *testing.T) {
	a, b := fa.Lit(fa.One('a')), fa.Lit(fa.One('b'))
	either := fa.Alt(a, b)
	given := make([]fa.Expr[rune], 0, 12)
	given = append(given, fa.Star(either), a)
	for range 10 {
		given = append(given, either)
	}
	rules := []fa.Rule[rune]{{Expr: fa.Cat(given...), Name: "tenth from the end"}}

	_, err := fa.BuildWith(rules, fa.Split(fa.Sets(rules)), 100)

	if err == nil || err.Error() != "the patterns need more than 100 states to match together" {
		t.Errorf("got %v; want the budget refused", err)
	}
}

func TestEdgesJoinTheClassesThatShareATarget(t *testing.T) {
	d := build(t, []fa.Rule[rune]{{Name: "one", Expr: fa.Alt(
		fa.Lit(fa.Text("ab")),
		fa.Lit(fa.Span('0', '9')),
		fa.Cat(fa.Lit(fa.One('c')), fa.Lit(fa.One('c'))),
	)}}).Minimise()

	var got []string
	for to, set := range d.Edges(d.Start()) {
		got = append(got, fmt.Sprintf("%s->%d", set, to))
	}

	if then := []string{"0-9ab->2", "c->3"}; !reflect.DeepEqual(got, then) {
		t.Errorf("the edges from the start are %q; want %q", got, then)
	}
}
