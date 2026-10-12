package lex

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sort"
	"unicode/utf8"
)

type Range struct {
	Lo, Hi rune
	Class  int
}

type Act struct {
	Emit int
	Do   int
	Goto int
	Push int
	Pop  bool
}

type Scope struct {
	Name   string
	Start  int
	Next   [][]int
	Accept []int
	Acts   []Act
}

type Spec struct {
	Ranges []Range
	Scopes []Scope
	Tokens []string
	Does   []string
}

func (s *Spec) Token(kind int) string {
	if kind < 1 || kind > len(s.Tokens) {
		return ""
	}

	return s.Tokens[kind-1]
}

type Token struct {
	Kind int
	Text string
	Do   int
	At   Pos
	Err  error
}

func (t Token) End() int { return t.At.Offset + len(t.Text) }

type Pos struct {
	Offset int
	Line   int
	Col    int
}

func (p Pos) after(text []byte) Pos {
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

func (s *Spec) class(r rune) int {
	at := sort.Search(len(s.Ranges), func(i int) bool { return s.Ranges[i].Hi >= r })

	if at < len(s.Ranges) && s.Ranges[at].Lo <= r {
		return s.Ranges[at].Class
	}

	return 0
}

type lexer struct {
	ctx   context.Context
	spec  *Spec
	r     io.Reader
	in    bytes.Buffer
	eof   bool
	pos   Pos
	scope int
	stack []int
	quiet [][]bool
	items chan Token
}

func Lex(ctx context.Context, spec *Spec, r io.Reader, scope string) (<-chan Token, error) {
	at := 0

	for i := range spec.Scopes {
		if spec.Scopes[i].Name == scope {
			at = i + 1
		}
	}

	if at == 0 {
		return nil, fmt.Errorf("no scope called %s", scope)
	}

	l := &lexer{ctx: ctx, spec: spec, r: r, pos: Pos{Line: 1, Col: 1}, scope: at, quiet: quiet(spec), items: make(chan Token, 64)}

	go l.run()

	return l.items, nil
}

func (l *lexer) run() {
	defer close(l.items)

	for l.match() {
	}
}

func (l *lexer) emit(t Token) bool {
	select {
	case l.items <- t:
		return true
	case <-l.ctx.Done():
		return false
	}
}

func (l *lexer) errorf(at Pos, format string, args ...any) bool {
	l.emit(Token{At: at, Err: fmt.Errorf(format, args...)})

	return false
}

func (l *lexer) match() bool {
	s := &l.spec.Scopes[l.scope-1]
	state, rule, back, backState := s.Start, 0, 0, 0
	at, first, read, dropped := l.pos, rune(0), 0, false

	for {
		buf := l.in.Bytes()

		if !l.eof && !utf8.FullRune(buf[read:]) {
			drop, dropState := back, backState

			if rule == 0 {
				drop, dropState = read, state
			}

			if drop > 0 && l.quiet[l.scope-1][dropState] {
				l.pos = l.pos.after(l.in.Next(drop))
				read, back, dropped = read-drop, 0, true
			}

			if err := l.fill(); err != nil {
				return l.errorf(l.pos, "%v", err)
			}

			continue
		}

		if read == len(buf) {
			break
		}

		r, w := utf8.DecodeRune(buf[read:])

		if read == 0 && !dropped {
			first = r
		}

		next := s.Next[state][l.spec.class(r)]

		if next == 0 {
			break
		}

		state, read = next, read+w

		if a := s.Accept[state]; a != 0 {
			rule, back, backState = a, read, state
		}
	}

	if l.in.Len() == 0 && !dropped {
		if len(l.stack) > 0 {
			return l.errorf(at, "the input ends inside %s", s.Name)
		}

		return false
	}

	if rule == 0 {
		return l.errorf(at, "%s: no rule reads %q", s.Name, first)
	}

	act := s.Acts[rule-1]
	text := l.in.Next(back)
	l.pos = l.pos.after(text)

	if act.Push != 0 {
		n := len(l.stack)
		l.stack = append(l.stack[:n:n], act.Push)
	}

	switch n := len(l.stack); {
	case act.Pop && n == 0:
		return l.errorf(at, "%s: returns with nothing to return to", s.Name)
	case act.Pop:
		l.scope, l.stack = l.stack[n-1], l.stack[:n-1]
	case act.Goto != 0:
		l.scope = act.Goto
	}

	if act.Emit == 0 {
		return true
	}

	return l.emit(Token{Kind: act.Emit, Do: act.Do, Text: string(text), At: at})
}

func (l *lexer) fill() error {
	l.in.Grow(4096)
	p := l.in.AvailableBuffer()
	n, err := l.r.Read(p[:cap(p)])
	l.in.Write(p[:n])

	if err == io.EOF {
		l.eof = true

		return nil
	}

	return err
}

func quiet(spec *Spec) [][]bool {
	out := make([][]bool, len(spec.Scopes))

	for i, s := range spec.Scopes {
		loud := make([]bool, len(s.Next))

		for st, a := range s.Accept {
			loud[st] = a != 0 && s.Acts[a-1].Emit != 0
		}

		for changed := true; changed; {
			changed = false

			for st := 1; st < len(s.Next); st++ {
				if loud[st] {
					continue
				}

				for _, to := range s.Next[st] {
					if to != 0 && loud[to] {
						loud[st], changed = true, true

						break
					}
				}
			}
		}

		out[i] = make([]bool, len(s.Next))

		for st := 1; st < len(s.Next); st++ {
			out[i][st] = !loud[st]
		}
	}

	return out
}
