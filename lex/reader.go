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

type Reader struct {
	run  *Lexer
	from string
	err  error
}

func Open(spec *Spec, in io.Reader, scope, from string) (*Reader, error) {
	run, err := New(spec, &terminated{r: in, ended: true}, scope)

	if err != nil {
		return nil, err
	}

	return &Reader{run: run, from: from}, nil
}

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

func (r *Reader) Next() (Token, bool) {
	tok, ok := r.run.Next()

	if !ok {
		if err := r.run.Err(); err != nil {
			r.Fail("%w", err)
		}
	}

	return tok, ok
}

func (r *Reader) Depth() int { return r.run.Depth() }

func (r *Reader) Line() int { return r.run.pos.Line }

func (r *Reader) From() string { return r.from }

func (r *Reader) Name() string {
	base := filepath.Base(r.from)

	if dot := strings.LastIndexByte(base, '.'); dot > 0 {
		base = base[:dot]
	}

	return base
}

func (r *Reader) Fail(f string, a ...any) { r.FailAt(r.Line(), f, a...) }

func (r *Reader) FailAt(line int, f string, a ...any) {
	if r.err == nil {
		r.err = &Error{File: r.from, Line: line, Err: fmt.Errorf(f, a...)}
	}
}

func (r *Reader) Err() error { return r.err }
