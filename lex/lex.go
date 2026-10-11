package lex

import (
	"bufio"
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
	Kind  int
	Text  string
	Do    int
	Start int
	End   int

	At Where
}

type Where struct {
	File string
	Line int
	Col  int
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
	at    int
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

	l := &Lexer{spec: spec, scope: at}
	l.sc = bufio.NewScanner(r)
	l.sc.Buffer(make([]byte, 4096), 1<<20)
	l.sc.Split(l.split)

	return l, nil
}

func (l *Lexer) split(data []byte, atEOF bool) (int, []byte, error) {
	done := 0

	for {
		if len(data) == done && atEOF {
			l.at += done

			return done, nil, nil
		}

		adv, tok, err := l.one(data[done:], atEOF)

		if err != nil {
			return 0, nil, err
		}

		if adv == 0 {
			l.at += done

			return done, nil, nil
		}

		done += adv

		if tok != nil {
			l.tok.Start += l.at + done - adv
			l.tok.End += l.at + done - adv
			l.at += done

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

	l.tok = Token{Kind: act.Emit, Do: act.Do, Text: string(text), Start: 0, End: back}

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
