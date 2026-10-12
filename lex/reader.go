package lex

import (
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"
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
	lines   []int
	at      int

	began, stmt      int
	started, content bool

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

	lines := []int{0}

	for i := range len(src) {
		if src[i] == '\n' {
			lines = append(lines, i+1)
		}
	}

	return &Reader{run: run, src: src, from: from, newline: newline, lines: lines}, nil
}

func (r *Reader) Next() (Token, bool) {
	tok, ok := r.run.Next()

	if !ok {
		if err := r.run.Err(); err != nil {
			r.Fail("%w", err)
		}

		return tok, false
	}

	if !r.started {
		r.began, r.started = tok.Start, true
	}

	r.at = tok.End

	if tok.Kind == r.newline {
		if r.content {
			r.stmt = r.began
		}

		r.started, r.content = false, false
	} else {
		r.content = true
	}

	tok.At = r.Where(tok.Start)

	return tok, true
}

func (r *Reader) Where(offset int) Where {
	line := r.line(offset)

	return Where{File: r.from, Line: line + 1, Col: utf8.RuneCountInString(r.src[r.lines[line]:offset]) + 1}
}

func (r *Reader) line(offset int) int {
	return sort.Search(len(r.lines), func(i int) bool { return r.lines[i] > offset }) - 1
}

func (r *Reader) Skip() { r.content = false }

func (r *Reader) Depth() int { return r.run.Depth() }

func (r *Reader) Line() int { return r.line(r.at) + 1 }

func (r *Reader) Statement() int { return r.line(r.stmt) + 1 }

func (r *Reader) From() string { return r.from }

func (r *Reader) Name() string {
	base := filepath.Base(r.from)

	if dot := strings.LastIndexByte(base, '.'); dot > 0 {
		base = base[:dot]
	}

	return base
}

func (r *Reader) Source(line int) string {
	if line < 1 || line >= len(r.lines) {
		return ""
	}

	return strings.TrimSpace(r.src[r.lines[line-1] : r.lines[line]-1])
}

func (r *Reader) Fail(f string, a ...any) { r.FailAt(r.Line(), f, a...) }

func (r *Reader) FailAt(line int, f string, a ...any) {
	if r.err == nil {
		r.err = &Error{File: r.from, Line: line, Err: fmt.Errorf(f, a...)}
	}
}

func (r *Reader) Err() error { return r.err }
