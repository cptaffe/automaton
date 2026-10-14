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

type State struct {
	Next   []int
	Accept int
}

type Scope struct {
	Name   string
	Start  int
	States []State
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
	at := l.pos
	m, err := l.scan(s)

	if err != nil {
		return l.errorf(l.pos, "%w", err)
	}

	if l.in.Len() == 0 && !m.dropped {
		if len(l.stack) > 0 {
			return l.errorf(at, "the input ends inside %s", s.Name)
		}

		return false
	}

	if m.rule == 0 {
		return l.errorf(at, "%s: no rule reads %q", s.Name, m.first)
	}

	act := s.Acts[m.rule-1]
	text := l.in.Next(m.back)
	l.pos = l.pos.after(text)

	if !l.apply(s, act, at) {
		return false
	}

	if act.Emit == 0 {
		return true
	}

	return l.emit(Token{Kind: act.Emit, Do: act.Do, Text: string(text), At: at})
}

type attempt struct {
	state     int
	read      int
	rule      int
	back      int
	backState int
	first     rune
	dropped   bool
}

func (m *attempt) step(next, w, accept int) {
	m.state, m.read = next, m.read+w

	if accept != 0 {
		m.rule, m.back, m.backState = accept, m.read, next
	}
}

func (m *attempt) droppable() (n, state int) {
	if m.rule == 0 {
		return m.read, m.state
	}

	return m.back, m.backState
}

func (l *lexer) scan(s *Scope) (attempt, error) {
	m := attempt{state: s.Start}

	for {
		buf := l.in.Bytes()

		if !l.eof && !utf8.FullRune(buf[m.read:]) {
			l.drop(&m)

			if err := l.fill(); err != nil {
				return m, err
			}

			continue
		}

		if m.read == len(buf) {
			return m, nil
		}

		r, w := utf8.DecodeRune(buf[m.read:])

		if m.read == 0 && !m.dropped {
			m.first = r
		}

		next := s.States[m.state].Next[l.spec.class(r)]

		if next == 0 {
			return m, nil
		}

		m.step(next, w, s.States[next].Accept)
	}
}

func (l *lexer) drop(m *attempt) {
	n, state := m.droppable()

	if n > 0 && l.quiet[l.scope-1][state] {
		l.pos = l.pos.after(l.in.Next(n))
		m.read, m.back, m.dropped = m.read-n, 0, true
	}
}

func (l *lexer) apply(s *Scope, act Act, at Pos) bool {
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

	return true
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

	for i := range spec.Scopes {
		out[i] = spec.Scopes[i].quiet()
	}

	return out
}

func (s *Scope) quiet() []bool {
	loud := make([]bool, len(s.States))

	for st, state := range s.States {
		loud[st] = state.Accept != 0 && s.Acts[state.Accept-1].Emit != 0
	}

	for changed := true; changed; {
		changed = false

		for st := 1; st < len(s.States); st++ {
			if !loud[st] && s.reaches(st, loud) {
				loud[st], changed = true, true
			}
		}
	}

	out := make([]bool, len(s.States))

	for st := 1; st < len(s.States); st++ {
		out[st] = !loud[st]
	}

	return out
}

func (s *Scope) reaches(st int, in []bool) bool {
	for _, to := range s.States[st].Next {
		if to != 0 && in[to] {
			return true
		}
	}

	return false
}
