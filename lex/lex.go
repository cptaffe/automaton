package lex

import (
	"io"
	"slices"
	"unicode/utf8"
)

const EOF rune = -1

type Token[K ~int] struct {
	Kind K
	Text string
	Do   int
	At   Pos
	Err  error
}

func (t Token[K]) End() int { return t.At.Offset + len(t.Text) }

type Pos struct {
	Offset int
	Line   int
	Col    int
}

func (p Pos) After(text []byte) Pos {
	p.Offset += len(text)

	for len(text) > 0 {
		r, w := utf8.DecodeRune(text)
		text = text[w:]

		if r == '\n' {
			p.Line, p.Col = p.Line+1, 1
		} else {
			p.Col++
		}
	}

	return p
}

type Range struct{ Lo, Hi rune }

func In(rs []Range, r rune) bool {
	at, found := slices.BinarySearchFunc(rs, r, func(x Range, r rune) int {
		switch {
		case x.Hi < r:
			return -1
		case x.Lo > r:
			return 1
		}

		return 0
	})

	return found && at < len(rs)
}

type Input struct {
	r     io.Reader
	buf   []byte
	read  int
	eof   bool
	err   error
	at    Pos
	start Pos
	first rune
	quiet bool
	mark  mark
}

type mark struct {
	rule  int
	read  int
	quiet bool
}

func NewInput(r io.Reader) Input { return Input{r: r, at: Pos{Line: 1, Col: 1}} }

func (in *Input) Next() rune {
	if in.read < len(in.buf) {
		if b := in.buf[in.read]; b < utf8.RuneSelf {
			in.read++

			return rune(b)
		}
	}

	return in.next()
}

func (in *Input) next() rune {
	for !in.eof && !utf8.FullRune(in.buf[in.read:]) {
		in.fill()
	}

	if in.read == len(in.buf) {
		return EOF
	}

	r, w := utf8.DecodeRune(in.buf[in.read:])
	in.read += w

	return r
}

func (in *Input) Quiet(q bool) { in.quiet = q }

func (in *Input) Mark(rule int) { in.mark = mark{rule: rule, read: in.read, quiet: in.quiet} }

func (in *Input) Begin() bool {
	in.read, in.mark, in.quiet, in.start = 0, mark{}, false, in.at

	for !in.eof && !utf8.FullRune(in.buf) {
		in.fill()
	}

	if len(in.buf) == 0 {
		return true
	}

	in.first, _ = utf8.DecodeRune(in.buf)

	return false
}

func (in *Input) First() rune { return in.first }

func (in *Input) Start() Pos { return in.start }

func (in *Input) Back() int {
	in.read = in.mark.read

	return in.mark.rule
}

func (in *Input) Take() (string, Pos) {
	s := string(in.buf[:in.read])
	in.Skip()

	return s, in.start
}

func (in *Input) Skip() {
	in.at = in.at.After(in.buf[:in.read])
	in.buf, in.read = in.buf[in.read:], 0
}

func (in *Input) At() Pos { return in.at }

func (in *Input) Err() error { return in.err }

func (in *Input) fill() {
	in.drop()

	if len(in.buf) == cap(in.buf) {
		in.buf = slices.Grow(in.buf, 4096)
	}

	n, err := in.r.Read(in.buf[len(in.buf):cap(in.buf)])
	in.buf = in.buf[:len(in.buf)+n]

	switch {
	case err == io.EOF:
		in.eof = true
	case err != nil:
		in.eof, in.err = true, err
	}
}

func (in *Input) drop() {
	n := in.read

	switch {
	case in.mark.rule != 0 && in.mark.quiet:
		n = in.mark.read
	case in.mark.rule != 0 || !in.quiet:
		return
	}

	if n == 0 {
		return
	}

	in.at = in.at.After(in.buf[:n])
	in.buf = append(in.buf[:0], in.buf[n:]...)
	in.read -= n
	in.mark.read = max(in.mark.read-n, 0)
}
