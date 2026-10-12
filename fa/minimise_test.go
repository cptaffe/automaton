package fa

import (
	"testing"
)

func TestMinimisingDoesNotChangeWhatIsRead(t *testing.T) {
	lit := func(s string) Expr[rune] {
		var out Expr[rune]
		first := true

		for _, r := range s {
			if first {
				out, first = Lit(One(r)), false

				continue
			}

			out = Cat(out, Lit(One(r)))
		}

		return out
	}

	alt := func(ss ...string) Expr[rune] {
		var out Expr[rune]

		for i, s := range ss {
			if i == 0 {
				out = lit(s)

				continue
			}

			out = Alt(out, lit(s))
		}

		return out
	}

	for _, c := range []struct {
		name  string
		rules []Rule[rune]
		runes string
	}{
		{

			"punctuators",
			[]Rule[rune]{{Name: "op", Expr: alt("<<=", ">>=", "++", "--", "<<", ">>",
				"<=", ">=", "+", "-", "<", ">", "(", ")", "&", "=")}},
			"<>+-()&=",
		},
		{
			"words and a prefix of one",
			[]Rule[rune]{
				{Name: "kw", Expr: alt("if", "in", "int")},
				{Name: "id", Expr: Plus(Lit(Span('a', 'z')))},
			},
			"ifnta",
		},
		{

			"punctuators beside names and keywords",
			[]Rule[rune]{
				{Name: "op", Expr: alt("<<=", ">>=", "++", "--", "<<", ">>", "<=", ">=",
					"==", "!=", "&&", "||", "+=", "-=", "->", "+", "-", "*", "/", "&",
					"|", "^", "~", "!", "<", ">", "=", "(", ")", "[", "]", "{", "}",
					";", ",", ".", "?", ":", "@")},
				{Name: "kw", Expr: alt("if", "else", "int", "u8", "for", "do", "return")},
				{Name: "id", Expr: Cat(Lit(Span('a', 'z')), Star(Lit(Span('a', 'z'))))},
				{Name: "num", Expr: Plus(Lit(Span('0', '9')))},
			},
			"<>+-()&=;a0i",
		},
		{
			"one rule that continues and one that does not",
			[]Rule[rune]{
				{Name: "a", Expr: alt("x", "xy")},
				{Name: "b", Expr: lit("z")},
			},
			"xyz",
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			full, err := Build(c.rules)

			if err != nil {
				t.Fatal(err)
			}

			small := full.Minimise()

			if small.States() > full.States() {
				t.Errorf("minimising grew the machine, %d states to %d", full.States(), small.States())
			}

			var walk func(at string, depth int)

			walk = func(at string, depth int) {
				a, b := full.Start(), small.Start()

				for _, r := range at {
					if a != NoState {
						a = full.Step(a, r)
					}

					if b != NoState {
						b = small.Step(b, r)
					}
				}

				if (a == NoState) != (b == NoState) {
					t.Fatalf("%q: one machine reads it and the other does not", at)
				}

				if a != NoState && full.Accept(a) != small.Accept(b) {
					t.Fatalf("%q: the machines accept it as different rules, %d and %d",
						at, full.Accept(a), small.Accept(b))
				}

				if depth == 0 {
					return
				}

				for _, r := range c.runes {
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
	if eq[0] != 0 || eq[2] != eq[3] || eq[1] == eq[2] || eq[4] == eq[2] {
		t.Errorf("classes %v: 2 and 3 are alike, and nothing else", eq)
	}
	to := Reachable(next, eq, 1)
	if to[5] != 0 || to[1] != 1 || to[2] != to[3] || to[4] == 0 {
		t.Errorf("reachable %v: 5 is never reached, and 2 and 3 are one state", to)
	}
}
