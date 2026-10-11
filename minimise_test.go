package automaton

import (
	"testing"
)

func TestMinimisingDoesNotChangeWhatIsRead(t *testing.T) {
	lit := func(s string) Expr {
		var out Expr
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

	alt := func(ss ...string) Expr {
		var out Expr

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
		rules []Rule
		runes string
	}{
		{

			"punctuators",
			[]Rule{{Name: "op", Expr: alt("<<=", ">>=", "++", "--", "<<", ">>",
				"<=", ">=", "+", "-", "<", ">", "(", ")", "&", "=")}},
			"<>+-()&=",
		},
		{
			"words and a prefix of one",
			[]Rule{
				{Name: "kw", Expr: alt("if", "in", "int")},
				{Name: "id", Expr: Plus(Lit(Span('a', 'z')))},
			},
			"ifnta",
		},
		{

			"punctuators beside names and keywords",
			[]Rule{
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
			[]Rule{
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
