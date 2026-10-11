package automaton

type kind int

const (
	lit kind = iota
	cat
	alt
	star
	plus
	empty
)

type Expr struct {
	kind kind
	set  Set
	sub  []Expr
}

func Lit(s Set) Expr { return Expr{kind: lit, set: s} }

func Empty() Expr { return Expr{kind: empty} }

func Cat(e ...Expr) Expr {
	switch len(e) {
	case 0:
		return Empty()
	case 1:
		return e[0]
	}

	return Expr{kind: cat, sub: e}
}

func Alt(e ...Expr) Expr {
	switch len(e) {
	case 0:
		return Empty()
	case 1:
		return e[0]
	}

	var one Set

	for _, s := range e {
		got, ok := s.Single()

		if !ok {
			return Expr{kind: alt, sub: e}
		}

		one = one.Union(got)
	}

	return Lit(one)
}

func Star(e Expr) Expr { return Expr{kind: star, sub: []Expr{e}} }

func Plus(e Expr) Expr { return Expr{kind: plus, sub: []Expr{e}} }

func Opt(e Expr) Expr { return Alt(e, Empty()) }

func (e Expr) Single() (Set, bool) {
	if e.kind == lit {
		return e.set, true
	}

	return Set{}, false
}

func (e Expr) sets(out []Set) []Set {
	if e.kind == lit {
		return append(out, e.set)
	}

	for _, s := range e.sub {
		out = s.sets(out)
	}

	return out
}
