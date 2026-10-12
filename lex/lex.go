package lex

import (
	"bufio"
	"fmt"
	"io"
	"math"
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
}

func (t Token) End() int { return t.At.Offset + len(t.Text) }

type Pos struct {
	Offset int
	Line   int
	Col    int
}

func (p Pos) after(text []byte) Pos {
	p.Offset += len(text)

	for _, r := range string(text) {
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

type Lexer struct {
	spec  *Spec
	sc    *bufio.Scanner
	scope int
	stack []int
	pos   Pos
	tok   Token
	err   error
}

func New(spec *Spec, r io.Reader, scope string) (*Lexer, error) {
	at := 0

	for i := range spec.Scopes {
		if spec.Scopes[i].Name == scope {
			at = i + 1
		}
	}

	if at == 0 {
		return nil, fmt.Errorf("no scope called %s", scope)
	}

	l := &Lexer{spec: spec, scope: at, pos: Pos{Line: 1, Col: 1}}
	l.sc = bufio.NewScanner(r)
	l.sc.Buffer(make([]byte, 4096), math.MaxInt32)
	l.sc.Split(l.split)

	return l, nil
}

func (l *Lexer) split(data []byte, atEOF bool) (int, []byte, error) {
	done := 0

	for {
		if len(data) == done && atEOF {
			return done, nil, nil
		}

		adv, tok, err := l.one(data[done:], atEOF)

		if err != nil {
			return 0, nil, err
		}

		if adv == 0 {
			return done, nil, nil
		}

		done += adv

		if tok != nil {
			return done, tok, nil
		}
	}
}

func (l *Lexer) one(data []byte, atEOF bool) (int, []byte, error) {
	if len(data) == 0 {
		return 0, nil, nil
	}

	s := &l.spec.Scopes[l.scope-1]
	state, rule, back := s.Start, 0, 0
	read := 0

	for read < len(data) {
		r, w := utf8.DecodeRune(data[read:])

		if r == utf8.RuneError && w <= 1 && !atEOF && len(data)-read < utf8.UTFMax {
			return 0, nil, nil
		}

		state = s.Next[state][l.spec.class(r)]

		if state == 0 {
			break
		}

		read += w

		if a := s.Accept[state]; a != 0 {
			rule, back = a, read
		}

		if read == len(data) && !atEOF {
			return 0, nil, nil
		}
	}

	if rule == 0 {
		r, _ := utf8.DecodeRune(data)

		return 0, nil, fmt.Errorf("%s: no rule reads %q", s.Name, r)
	}

	act := s.Acts[rule-1]
	text := data[:back]
	at := l.pos
	l.pos = at.after(text)

	if act.Push != 0 {
		n := len(l.stack)
		l.stack = append(l.stack[:n:n], act.Push)
	}

	switch n := len(l.stack); {
	case act.Pop && n == 0:
		return 0, nil, fmt.Errorf("%s: returns with nothing to return to", s.Name)
	case act.Pop:
		l.scope, l.stack = l.stack[n-1], l.stack[:n-1]
	case act.Goto != 0:
		l.scope = act.Goto
	}

	if act.Emit == 0 {
		return back, nil, nil
	}

	l.tok = Token{Kind: act.Emit, Do: act.Do, Text: string(text), At: at}

	return back, text, nil
}

func (l *Lexer) Next() (Token, bool) {
	if !l.sc.Scan() {
		l.err = l.sc.Err()

		return Token{}, false
	}

	return l.tok, true
}

func (l *Lexer) Err() error { return l.err }

func (l *Lexer) Scope() string { return l.spec.Scopes[l.scope-1].Name }

func (l *Lexer) Depth() int { return len(l.stack) }
