package seq

import (
	"fmt"

	"github.com/cptaffe/automaton/fa"
)

type Expr interface{ expr() }

type Cat []Expr
type Alt []Expr
type Star struct{ Of Expr }
type Plus struct{ Of Expr }
type Opt struct{ Of Expr }
type Minus struct{ From, Take Expr }

func (Atom) expr()  {}
func (Cat) expr()   {}
func (Alt) expr()   {}
func (Star) expr()  {}
func (Plus) expr()  {}
func (Opt) expr()   {}
func (Minus) expr() {}

func bind(e Expr, a *Alphabet) (Expr, error) {
	each := func(es []Expr) ([]Expr, error) {
		out := make([]Expr, len(es))
		for i, x := range es {
			got, err := bind(x, a)
			if err != nil {
				return nil, err
			}
			out[i] = got
		}
		return out, nil
	}
	one := func(x Expr, wrap func(Expr) Expr) (Expr, error) {
		got, err := bind(x, a)
		if err != nil {
			return nil, err
		}
		return wrap(got), nil
	}
	switch e := e.(type) {
	case Atom:
		return resolve(e, a)
	case Cat:
		got, err := each(e)
		return Cat(got), err
	case Alt:
		got, err := each(e)
		return Alt(got), err
	case Star:
		return one(e.Of, func(x Expr) Expr { return Star{x} })
	case Plus:
		return one(e.Of, func(x Expr) Expr { return Plus{x} })
	case Opt:
		return one(e.Of, func(x Expr) Expr { return Opt{x} })
	case Minus:
		got, err := each([]Expr{e.From, e.Take})
		if err != nil {
			return nil, err
		}
		return Minus{got[0], got[1]}, nil
	}
	return e, nil
}

func sets(e Expr, out []fa.Set[Sym]) []fa.Set[Sym] {
	switch e := e.(type) {
	case Atom:
		return append(out, e.set())
	case Cat:
		for _, x := range e {
			out = sets(x, out)
		}
	case Alt:
		for _, x := range e {
			out = sets(x, out)
		}
	case Star:
		return sets(e.Of, out)
	case Plus:
		return sets(e.Of, out)
	case Opt:
		return sets(e.Of, out)
	case Minus:
		if s, _, err := setOf(e); err == nil {
			return append(out, s)
		}
	}
	return out
}

func lower(e Expr) (fa.Expr[Sym], error) {
	each := func(es []Expr) ([]fa.Expr[Sym], error) {
		out := make([]fa.Expr[Sym], 0, len(es))
		for _, x := range es {
			got, err := lower(x)
			if err != nil {
				return nil, err
			}
			out = append(out, got)
		}
		return out, nil
	}
	switch e := e.(type) {
	case Atom, Minus:
		s, weak, err := setOf(e)
		if err != nil {
			return fa.Expr[Sym]{}, err
		}
		if weak {
			return fa.Fallback(s), nil
		}
		return fa.Lit(s), nil
	case Cat:
		got, err := each(e)
		return fa.Cat(got...), err
	case Alt:
		got, err := each(e)
		return fa.Alt(got...), err
	case Star:
		got, err := lower(e.Of)
		return fa.Star(got), err
	case Plus:
		got, err := lower(e.Of)
		return fa.Plus(got), err
	case Opt:
		got, err := lower(e.Of)
		return fa.Opt(got), err
	}
	return fa.Expr[Sym]{}, fmt.Errorf("%T is not a pattern", e)
}

func setOf(e Expr) (fa.Set[Sym], bool, error) {
	switch e := e.(type) {
	case Atom:
		return e.set(), e.Any, nil
	case Alt:
		var out fa.Set[Sym]
		weak := false
		for _, x := range e {
			s, w, err := setOf(x)
			if err != nil {
				return fa.Set[Sym]{}, false, err
			}
			out, weak = out.Union(s), weak || w
		}
		return out, weak, nil
	case Minus:
		from, weak, err := setOf(e.From)
		if err != nil {
			return fa.Set[Sym]{}, false, err
		}
		take, _, err := setOf(e.Take)
		if err != nil {
			return fa.Set[Sym]{}, false, err
		}
		left := from.Without(take)
		if left.Empty() {
			return fa.Set[Sym]{}, false, fmt.Errorf("%s leaves no keys", text(e))
		}
		return left, weak, nil
	}
	return fa.Set[Sym]{}, false, fmt.Errorf("%s is not a set of keys", text(e))
}

func text(e Expr) string {
	switch e := e.(type) {
	case Atom:
		return e.Text
	case Minus:
		return text(e.From) + " - " + text(e.Take)
	}
	return "a sequence or repeat"
}

func sequence(e Expr) ([]Expr, bool) {
	items, ok := e.(Cat)
	if !ok {
		items = Cat{e}
	}
	for _, x := range items {
		switch x := x.(type) {
		case Atom:
		case Plus:
			if _, ok := x.Of.(Atom); !ok {
				return nil, false
			}
		default:
			return nil, false
		}
	}
	return items, true
}
