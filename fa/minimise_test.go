package fa

import (
	"reflect"
	"testing"
)

func word(s string) Expr[rune] {
	e := make([]Expr[rune], 0, len(s))

	for _, r := range s {
		e = append(e, Lit(One(r)))
	}

	return Cat(e...)
}

func either(ss ...string) Expr[rune] {
	e := make([]Expr[rune], 0, len(ss))

	for _, s := range ss {
		e = append(e, word(s))
	}

	return Alt(e...)
}

type sizes struct{ full, small int }

func TestMinimisingDoesNotChangeWhatIsRead(t *testing.T) {
	for _, c := range []struct {
		name  string
		given []Rule[rune]
		when  string
		then  sizes
	}{
		{
			"punctuators",
			[]Rule[rune]{{Name: "op", Expr: either("<<=", ">>=", "++", "--", "<<", ">>",
				"<=", ">=", "+", "-", "<", ">", "(", ")", "&", "=")}},
			"<>+-()&=",
			sizes{17, 7},
		},
		{
			"words and a prefix of one",
			[]Rule[rune]{
				{Name: "kw", Expr: either("if", "in", "int")},
				{Name: "id", Expr: Plus(Lit(Span('a', 'z')))},
			},
			"ifnta",
			sizes{6, 5},
		},
		{
			"punctuators beside names and keywords",
			[]Rule[rune]{
				{Name: "op", Expr: either("<<=", ">>=", "++", "--", "<<", ">>", "<=", ">=",
					"==", "!=", "&&", "||", "+=", "-=", "->", "+", "-", "*", "/", "&",
					"|", "^", "~", "!", "<", ">", "=", "(", ")", "[", "]", "{", "}",
					";", ",", ".", "?", ":", "@")},
				{Name: "kw", Expr: either("if", "else", "int", "u8", "for", "do", "return")},
				{Name: "id", Expr: Cat(Lit(Span('a', 'z')), Star(Lit(Span('a', 'z'))))},
				{Name: "num", Expr: Plus(Lit(Span('0', '9')))},
			},
			"<>+-()&=;a0i",
			sizes{64, 27},
		},
		{
			"one rule that continues and one that does not",
			[]Rule[rune]{
				{Name: "a", Expr: either("x", "xy")},
				{Name: "b", Expr: word("z")},
			},
			"xyz",
			sizes{4, 4},
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			full, err := Build(c.given)

			if err != nil {
				t.Fatal(err)
			}

			small := full.Minimise()

			if got := (sizes{full.States(), small.States()}); got != c.then {
				t.Errorf("the machine has %d states and %d minimised; want %d and %d",
					got.full, got.small, c.then.full, c.then.small)
			}

			var walk func(at string, depth int)

			walk = func(at string, depth int) {
				a, b := full.Start(), small.Start()

				for _, r := range at {
					a, b = full.Step(a, r), small.Step(b, r)
				}

				if (a == NoState) != (b == NoState) {
					t.Fatalf("%q: one machine reads it and the other does not", at)
				}

				if a == NoState {
					return
				}

				if !reflect.DeepEqual(full.Done(a), small.Done(b)) || !reflect.DeepEqual(full.Going(a), small.Going(b)) {
					t.Fatalf("%q: done %v going %v, and minimised done %v going %v",
						at, full.Done(a), full.Going(a), small.Done(b), small.Going(b))
				}

				if depth == 0 {
					return
				}

				for _, r := range c.when {
					walk(at+string(r), depth-1)
				}
			}

			walk("", 4)
		})
	}
}

func TestEquivalentMergesWhatNothingTellsApart(t *testing.T) {
	next := [][]int{
		{0, 0},
		{2, 3},
		{4, 0},
		{4, 0},
		{0, 0},
		{0, 0},
	}
	sig := []string{"", "", "", "", "end", "unreached"}

	eq := Equivalent(next, sig)

	if then := []int{0, 1, 2, 2, 3, 4}; !reflect.DeepEqual(eq, then) {
		t.Errorf("the groups are %v; want %v, with only 2 and 3 alike", eq, then)
	}
}

func TestReachableDropsWhatTheStartNeverReaches(t *testing.T) {
	next := [][]int{
		{0, 0},
		{2, 3},
		{4, 0},
		{4, 0},
		{0, 0},
		{0, 0},
	}
	eq := []int{0, 1, 2, 2, 3, 4}

	to := Reachable(next, eq, 1)

	if then := []int{0, 1, 2, 2, 3, 0}; !reflect.DeepEqual(to, then) {
		t.Errorf("the states are %v; want %v, with 5 never reached", to, then)
	}
}
