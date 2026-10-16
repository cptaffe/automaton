package lex

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"
)

type Error struct {
	File string
	Line int
	Err  error
}

func (e *Error) Error() string { return fmt.Sprintf("%s:%d: %v", e.File, e.Line, e.Err) }

func (e *Error) Unwrap() error { return e.Err }

type Reader[K ~int] struct {
	toks Tokens[K]
	from string
	line int
	err  error
}

func NewReader[K ~int](toks Tokens[K], from string) *Reader[K] {
	return &Reader[K]{toks: toks, from: from, line: 1}
}

func Terminated(r io.Reader) io.Reader { return &terminated{r: r, ended: true} }

type terminated struct {
	r          io.Reader
	eof, ended bool
}

func (t *terminated) Read(p []byte) (int, error) {
	if t.eof {
		if t.ended || len(p) == 0 {
			return 0, io.EOF
		}

		p[0], t.ended = '\n', true

		return 1, io.EOF
	}

	n, err := t.r.Read(p)

	if n > 0 {
		t.ended = p[n-1] == '\n'
	}

	if err == io.EOF {
		t.eof, err = true, nil
	}

	return n, err
}

func (r *Reader[K]) Next() (Token[K], bool) {
	tok, ok := r.toks.Next()

	if !ok {
		return tok, false
	}

	if tok.Err != nil {
		r.line = tok.At.Line
		r.Fail("%w", tok.Err)

		return Token[K]{}, false
	}

	r.line = tok.At.Line + strings.Count(tok.Text, "\n")

	return tok, true
}

func (r *Reader[K]) Line() int { return r.line }

func (r *Reader[K]) From() string { return r.from }

func (r *Reader[K]) Name() string {
	base := filepath.Base(r.from)

	if dot := strings.LastIndexByte(base, '.'); dot > 0 {
		base = base[:dot]
	}

	return base
}

func (r *Reader[K]) Fail(f string, a ...any) { r.FailAt(r.Line(), f, a...) }

func (r *Reader[K]) FailAt(line int, f string, a ...any) {
	if r.err == nil {
		r.err = &Error{File: r.from, Line: line, Err: fmt.Errorf(f, a...)}
	}
}

func (r *Reader[K]) Err() error { return r.err }
