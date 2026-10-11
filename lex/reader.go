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
	src  string
	from string

	newline int

	line, col int

	began, stmt  int
	fresh, empty bool

	err error
}

func Open(spec *Spec, in io.Reader, scope, from string, newline int) (*Reader, error) {
	b, err := io.ReadAll(in)

	if err != nil {
		return nil, err
	}

	src := string(b)

	if src != "" && !strings.HasSuffix(src, "\n") {
		src += "\n"
	}

	run, err := New(spec, strings.NewReader(src), scope)

	if err != nil {
		return nil, err
	}

	return &Reader{
		run: run, src: src, from: from, newline: newline,
		line: 1, col: 1, began: 1, stmt: 1, fresh: true, empty: true,
	}, nil
}

func (r *Reader) Next() (Token, bool) {
	if r.fresh {
		r.began, r.fresh = r.line, false
	}

	tok, ok := r.run.Next()

	if !ok {
		if err := r.run.Err(); err != nil {
			r.Fail("%w", err)
		}

		return tok, false
	}

	at := Where{File: r.from, Line: r.line, Col: r.col}
	r.walk(tok.Text)

	if tok.Kind == r.newline {
		if !r.empty {
			r.stmt = r.began
		}

		r.fresh, r.empty = true, true
	} else {
		r.empty = false
	}

	tok.At = at

	return tok, true
}

func (r *Reader) Skip() { r.empty = true }

func (r *Reader) Depth() int { return r.run.Depth() }

func (r *Reader) walk(text string) {
	if n := strings.Count(text, "\n"); n > 0 {
		r.line += n
		r.col = len([]rune(text[strings.LastIndexByte(text, '\n')+1:])) + 1

		return
	}

	r.col += len([]rune(text))
}

func (r *Reader) Line() int { return r.line }

func (r *Reader) Statement() int { return r.stmt }

func (r *Reader) From() string { return r.from }

func (r *Reader) Name() string {
	base := filepath.Base(r.from)

	if dot := strings.LastIndexByte(base, '.'); dot > 0 {
		base = base[:dot]
	}

	return base
}

func (r *Reader) Source(line int) string {
	lines := strings.Split(r.src, "\n")

	if line < 1 || line > len(lines) {
		return ""
	}

	return strings.TrimSpace(lines[line-1])
}

func (r *Reader) Fail(f string, a ...any) { r.FailAt(r.line, f, a...) }

func (r *Reader) FailAt(line int, f string, a ...any) {
	if r.err == nil {
		r.err = &Error{File: r.from, Line: line, Err: fmt.Errorf(f, a...)}
	}
}

func (r *Reader) Err() error { return r.err }
