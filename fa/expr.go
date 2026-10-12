package fa

type kind int

const (
	empty kind = iota
	lit
	cat
	alt
	star
	plus
)

type Expr[S Symbol] struct {
	kind kind
	set  Set[S]
	sub  []Expr[S]
}

func Lit[S Symbol](s Set[S]) Expr[S] { return Expr[S]{kind: lit, set: s} }

func Empty[S Symbol]() Expr[S] { return Expr[S]{} }

func Cat[S Symbol](e ...Expr[S]) Expr[S] {
	switch len(e) {
	case 0:
		return Empty[S]()
	case 1:
		return e[0]
	}

	return Expr[S]{kind: cat, sub: e}
}

func Alt[S Symbol](e ...Expr[S]) Expr[S] {
	switch len(e) {
	case 0:
		return Empty[S]()
	case 1:
		return e[0]
	}

	var one Set[S]

	for _, s := range e {
		got, ok := s.Single()

		if !ok {
			return Expr[S]{kind: alt, sub: e}
		}

		one = one.Union(got)
	}

	return Lit(one)
}

func Star[S Symbol](e Expr[S]) Expr[S] { return Expr[S]{kind: star, sub: []Expr[S]{e}} }

func Plus[S Symbol](e Expr[S]) Expr[S] { return Expr[S]{kind: plus, sub: []Expr[S]{e}} }

func Opt[S Symbol](e Expr[S]) Expr[S] { return Alt(e, Empty[S]()) }

func (e Expr[S]) Single() (Set[S], bool) {
	if e.kind == lit {
		return e.set, true
	}

	return Set[S]{}, false
}

func (e Expr[S]) sets(out []Set[S]) []Set[S] {
	if e.kind == lit {
		return append(out, e.set)
	}

	for _, s := range e.sub {
		out = s.sets(out)
	}

	return out
}
