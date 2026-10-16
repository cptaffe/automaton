package lex

import (
	"fmt"
	"io"
	"iter"
)

type Tokens[K ~int] interface {
	Next() (Token[K], bool)
	All() iter.Seq[Token[K]]
}

type Emitter[K ~int, S any] struct {
	Input
	item  Token[K]
	ready bool
	stack []S
}

func NewEmitter[K ~int, S any](r io.Reader) Emitter[K, S] {
	return Emitter[K, S]{Input: NewInput(r)}
}

func (e *Emitter[K, S]) Ready() bool { return e.ready }

func (e *Emitter[K, S]) Item() (Token[K], bool) {
	t, ok := e.item, e.ready
	e.item, e.ready = Token[K]{}, false

	return t, ok
}

func (e *Emitter[K, S]) Emit(kind K, do int) {
	text, at := e.Take()
	e.item, e.ready = Token[K]{Kind: kind, Do: do, Text: text, At: at}, true
}

func (e *Emitter[K, S]) Errorf(at Pos, format string, args ...any) {
	e.item, e.ready = Token[K]{At: at, Err: fmt.Errorf(format, args...)}, true
}

func (e *Emitter[K, S]) Push(s S) { e.stack = append(e.stack, s) }

func (e *Emitter[K, S]) Pop(scope string) (S, bool) {
	n := len(e.stack)

	if n == 0 {
		var none S
		e.Errorf(e.Start(), "%s: returns with nothing to return to", scope)

		return none, false
	}

	s := e.stack[n-1]
	e.stack = e.stack[:n-1]

	return s, true
}

func (e *Emitter[K, S]) End(scope string) {
	switch {
	case e.Err() != nil:
		e.Errorf(e.At(), "%w", e.Err())
	case len(e.stack) > 0:
		e.Errorf(e.Start(), "the input ends inside %s", scope)
	}
}

func (e *Emitter[K, S]) Fail(scope string) {
	if e.Err() != nil {
		e.Errorf(e.At(), "%w", e.Err())

		return
	}

	e.Errorf(e.Start(), "%s: no rule reads %q", scope, e.First())
}
