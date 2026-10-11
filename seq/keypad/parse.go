package keypad

//go:generate go run ../../cmd/genlex -go internal/scan/scan.gen.go -package scan tree.lex
//go:generate go tool goyacc -l -p yy -o gram.go -v /dev/null gram.y

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/cptaffe/automaton/lexrun"
	"github.com/cptaffe/automaton/seq"
	"github.com/cptaffe/automaton/seq/keypad/internal/scan"
)

var Keys = []string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9", "*", "#"}

var Events = []string{"answered"}

const Answered = "answered"

func Alphabet(gs ...*seq.Grammar) *seq.Alphabet {
	var delays []string
	seen := map[string]bool{}
	for _, e := range Events {
		seen[e] = true
	}
	for _, g := range gs {
		for name := range g.Delays {
			if !seen[name] {
				seen[name] = true
				delays = append(delays, name)
			}
		}
	}
	sort.Strings(delays)
	return seq.NewAlphabet(Keys, append(append([]string(nil), Events...), delays...)...)
}

var classes = map[rune][]int{
	'N': {2, 3, 4, 5, 6, 7, 8, 9},
	'X': {0, 1, 2, 3, 4, 5, 6, 7, 8, 9},
}

var kinds = map[int]int{
	scan.TokNewline: NEWLINE,
	scan.TokColon:   COLON,
	scan.TokLparen:  LPAREN,
	scan.TokRparen:  RPAREN,
	scan.TokDots:    DOTS,
	scan.TokArrow:   ARROW,
	scan.TokHandoff: HANDOFF,
	scan.TokDelay:   DELAY,
	scan.TokSettles: SETTLES,
	scan.TokDo:      DO,
	scan.TokWhen:    WHEN,
	scan.TokAnd:     AND,
	scan.TokOr:      OR,
	scan.TokNot:     NOT,
	scan.TokSym:     SYM,
	scan.TokName:    IDENT,
}

type word struct {
	text string
	raw  string
	line int
}

type symv struct {
	c    seq.Class
	line int
}

type pat struct {
	elems []seq.Class
	text  string
	line  int
}

type lexer struct {
	*lexrun.Reader
	held    string
	tok     int
	tokText string
	tokLine int
	block   []string
	bare    bool
	g       *seq.Grammar
	cur     string
}

func Parse(r io.Reader, from string) (*seq.Grammar, error) {
	in, err := lexrun.Open(scan.Spec, r, "start", from, scan.TokNewline)
	if err != nil {
		return nil, err
	}
	l := &lexer{Reader: in, bare: true, g: &seq.Grammar{Name: in.Name()}}
	if yyParse(l) != 0 && l.Err() == nil {
		l.FailAt(l.tokLine, "%s", l.unexpected())
	}
	if err := l.Err(); err != nil {
		return nil, err
	}
	if err := seq.Check(l.g); err != nil {
		return nil, err
	}
	return l.g, nil
}

func (l *lexer) Lex(v *yySymType) int {
	t, text := l.lex(v)
	l.tok, l.tokText, l.tokLine = t, text, l.Line()
	return t
}

func (l *lexer) Error(string) {}

func (l *lexer) lex(v *yySymType) (int, string) {
	for {
		tok, ok := l.Next()
		if !ok {
			return 0, ""
		}
		switch tok.Kind {
		case scan.TokBlank, scan.TokDash:
			l.held += tok.Text
			l.Skip()
			continue
		case scan.TokComment:
			l.comment(tok.Text, l.bare)
			l.bare = false
			l.Skip()
			continue
		}
		held := l.held
		l.held = ""
		if tok.Kind != scan.TokNewline {
			l.bare = false
		}
		switch tok.Kind {
		case scan.TokNewline:
			if l.bare {
				l.block = nil
			}
			l.bare = true
			return NEWLINE, "a new line"
		case scan.TokDelay, scan.TokSettles:
			v.word = word{tok.Text, held + tok.Text, tok.At.Line}
		case scan.TokSym:
			c, ok := symbolOf(held + tok.Text)
			if !ok {
				l.Fail("%q is not a pattern symbol", tok.Text)
				return 0, ""
			}
			v.sym = symv{c, tok.At.Line}
			return SYM, tok.Text
		case scan.TokDeclaration:
			l.Fail("%s is not a declaration", tok.Text)
			return 0, ""
		case scan.TokName:
			v.word = word{tok.Text, held + tok.Text, tok.At.Line}
			return IDENT, tok.Text
		}
		return kinds[tok.Kind], tok.Text
	}
}

func symbolOf(raw string) (seq.Class, bool) {
	rs := []rune(raw)
	at := 0
	for at < len(rs) && (rs[at] == ' ' || rs[at] == '\t' || rs[at] == '-') {
		at++
	}
	if at >= len(rs) {
		return seq.Class{}, false
	}
	var c seq.Class
	if keys, ok := classes[rs[at]]; ok {
		for _, k := range keys {
			c.Syms = append(c.Syms, seq.Sym(k))
		}
	} else if s := strings.IndexRune("0123456789*#", rs[at]); s >= 0 {
		c.Syms = []seq.Sym{seq.Sym(s)}
	} else {
		return seq.Class{}, false
	}
	c.Text = string(rs[:at+1])
	if at+1 < len(rs) && rs[at+1] == '+' {
		c.Many = true
		c.Text += "+"
	}
	return c, true
}

func one(s symv) pat {
	return pat{elems: []seq.Class{s.c}, text: strings.TrimLeft(s.c.Text, " \t"), line: s.line}
}

func add(p pat, s symv) pat {
	return pat{elems: append(p.elems, s.c), text: p.text + s.c.Text, line: p.line}
}

func event(t word) symv {
	return symv{seq.Class{Event: t.text, Text: t.raw}, t.line}
}

func or(a, b seq.Cond) seq.Cond {
	if o, ok := a.(seq.Or); ok {
		return append(o, b)
	}
	return seq.Or{a, b}
}

func and(a, b seq.Cond) seq.Cond {
	if x, ok := a.(seq.And); ok {
		return append(x, b)
	}
	return seq.And{a, b}
}

func (l *lexer) unexpected() string {
	switch l.tok {
	case SYM:
		return fmt.Sprintf("%q is a pattern symbol, and one cannot stand here", l.tokText)
	case IDENT:
		return fmt.Sprintf("%q is a name, and one cannot stand here", l.tokText)
	}
	return lexrun.Unexpected(l.tok, l.tokText)
}

func (l *lexer) comment(text string, bol bool) {
	note := strings.TrimSpace(strings.TrimPrefix(text, ";"))
	if note == "" {
		return
	}
	if l.g.Note == "" && l.cur == "" && len(l.g.Rules) == 0 {
		l.g.Note = note
	}
	if bol {
		l.block = append(l.block, note)
	}
}

func (l *lexer) says() string {
	block := l.block
	l.block = nil
	for _, line := range block {
		if label, ok := strings.CutPrefix(line, "label:"); ok {
			return strings.TrimSpace(label)
		}
	}
	return ""
}

func (l *lexer) delay(line int, name, of string) {
	l.says()
	if l.cur != "" {
		l.FailAt(line, "%%delay %s %s belongs above every scope: a symbol lasts as long wherever it is read", name, of)
		return
	}
	d, err := time.ParseDuration(of)
	if err != nil {
		l.FailAt(line, "%%delay %s %s: %s is not a duration", name, of, of)
		return
	}
	if d <= 0 {
		l.FailAt(line, "%%delay %s %s: a wait of no time is not a wait", name, of)
		return
	}
	if was, ok := l.g.Delays[name]; ok && was != d {
		l.FailAt(line, "%%delay %s %s: it is already %s", name, of, was)
		return
	}
	if l.g.Delays == nil {
		l.g.Delays = map[string]time.Duration{}
	}
	l.g.Delays[name] = d
}

func (l *lexer) settles(line int, name string) {
	l.says()
	if l.cur == "" {
		l.FailAt(line, "%%settles %s is above every scope, and it settles one scope", name)
		return
	}
	if was, ok := l.g.Settles[l.cur]; ok && was != name {
		l.FailAt(line, "%%settles %s: %s already settles %s", name, was, l.cur)
		return
	}
	if l.g.Settles == nil {
		l.g.Settles = map[string]string{}
	}
	l.g.Settles[l.cur] = name
}

func (l *lexer) scope(name string) {
	l.says()
	l.cur = name
	for _, s := range l.g.Scopes {
		if s == name {
			return
		}
	}
	l.g.Scopes = append(l.g.Scopes, name)
}

func (l *lexer) rule(r seq.Rule) {
	label := l.says()
	if l.cur == "" {
		l.FailAt(r.Line, "%q is above every scope, where only a `...` rule can be", r.Pattern)
		return
	}
	if r.Action == "" && r.Goto == "" && r.Hands == "" {
		l.FailAt(r.Line, "%q has a pattern and nowhere to go", r.Pattern)
		return
	}
	if err := named(r.Action); err != nil {
		l.FailAt(r.Line, "%s", err)
		return
	}
	r.Scope, r.Label = l.cur, label
	l.g.Rules = append(l.g.Rules, r)
}

func (l *lexer) anyRule(a seq.Any) {
	label := l.says()
	if err := named(a.Action); err != nil {
		l.FailAt(a.Line, "%s", err)
		return
	}
	a.Scope, a.Label = l.cur, label
	l.g.Any = append(l.g.Any, a)
}

func named(a string) error {
	for _, r := range a {
		if (r >= 'a' && r <= 'z') || r == '-' {
			continue
		}
		return fmt.Errorf("the action %q is not lower case and hyphens", a)
	}
	return nil
}
