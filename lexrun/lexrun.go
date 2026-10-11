package lexrun

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

const (
	noClass = -1
	noState = -1
	noMatch = -1
)

const None = -1

type Act struct {
	Emit int
	Do   int
	Goto int
	Drop bool
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

	return noClass
}

type Lexer struct {
	spec  *Spec
	sc    *bufio.Scanner
	scope int
	at    int
	tok   Token
	err   error
}

func New(spec *Spec, r io.Reader, scope string) (*Lexer, error) {
	at := -1

	for i := range spec.Scopes {
		if spec.Scopes[i].Name == scope {
			at = i
		}
	}

	if at < 0 {
		return nil, fmt.Errorf("no scope called %s", scope)
	}

	l := &Lexer{spec: spec, scope: at}
	l.sc = bufio.NewScanner(r)
	l.sc.Buffer(make([]byte, 4096), 1<<20)
	l.sc.Split(l.split)

	return l, nil
}

func (l *Lexer) split(data []byte, atEOF bool) (int, []byte, error) {
	was, scope := l.at, l.scope
	done := 0

	give := func(n int, tok []byte) (int, []byte, error) {
		l.at = was + n

		return n, tok, nil
	}

	drop := func() (int, []byte, error) {
		l.at, l.scope = was, scope

		return 0, nil, nil
	}

	for {
		if len(data) == done && atEOF {
			return give(done, nil)
		}

		adv, tok, err := l.one(data[done:], atEOF)

		if err != nil {
			return 0, nil, err
		}

		if adv == 0 && tok == nil {
			if done > 0 && atEOF {
				return give(done, nil)
			}

			return drop()
		}

		done += adv

		if tok != nil {
			l.tok.Start += was + done - adv
			l.tok.End += was + done - adv

			return give(done, tok)
		}
	}
}

func (l *Lexer) one(data []byte, atEOF bool) (int, []byte, error) {
	if len(data) == 0 {
		return 0, nil, nil
	}

	s := &l.spec.Scopes[l.scope]
	state, rule, back := s.Start, noMatch, 0
	read := 0

	for {
		if read == len(data) {
			if !atEOF {
				return 0, nil, nil
			}

			break
		}

		r, w := utf8.DecodeRune(data[read:])

		if r == utf8.RuneError && w <= 1 && !atEOF && len(data)-read < utf8.UTFMax {
			return 0, nil, nil
		}

		c := l.spec.class(r)

		if c == noClass {
			break
		}

		to := s.Next[state][c]

		if to == noState {
			break
		}

		state, read = to, read+w

		if a := s.Accept[state]; a != noMatch {
			rule, back = a, read
		}
	}

	if rule == noMatch {
		r, _ := utf8.DecodeRune(data)

		return 0, nil, fmt.Errorf("%s: no rule reads %q", s.Name, r)
	}

	act := s.Acts[rule]
	text := data[:back]

	if act.Goto != None {
		l.scope = act.Goto
	}

	kind := act.Emit

	if act.Drop || kind == None {
		return back, nil, nil
	}

	l.tok = Token{Kind: kind, Do: act.Do, Text: string(text), Start: 0, End: back}

	return back, text, nil
}

func (l *Lexer) Next() (Token, bool) {
	if !l.sc.Scan() {
		l.err = l.sc.Err()

		return Token{Kind: noMatch}, false
	}

	return l.tok, true
}

func (l *Lexer) Err() error { return l.err }

func (l *Lexer) Scope() string { return l.spec.Scopes[l.scope].Name }
