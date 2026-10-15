package lex

import (
	"context"
	"fmt"
	"io"
)

type Emitter[K ~int, S any] struct {
	Input
	ctx   context.Context
	items chan Token[K]
	stack []S
}

func NewEmitter[K ~int, S any](ctx context.Context, r io.Reader) Emitter[K, S] {
	return Emitter[K, S]{Input: NewInput(r), ctx: ctx, items: make(chan Token[K], 64)}
}

func (e *Emitter[K, S]) Items() <-chan Token[K] { return e.items }

func (e *Emitter[K, S]) Close() { close(e.items) }

func (e *Emitter[K, S]) Emit(kind K, do int) bool {
	text, at := e.Take()

	return e.send(Token[K]{Kind: kind, Do: do, Text: text, At: at})
}

func (e *Emitter[K, S]) Errorf(at Pos, format string, args ...any) {
	e.send(Token[K]{At: at, Err: fmt.Errorf(format, args...)})
}

func (e *Emitter[K, S]) send(t Token[K]) bool {
	select {
	case e.items <- t:
		return true
	case <-e.ctx.Done():
		return false
	}
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
