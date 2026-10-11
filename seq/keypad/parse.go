package keypad

//go:generate go run ../../cmd/genlex -go internal/scan/scan.gen.go -package scan tree.lex
//go:generate go tool goyacc -l -p yy -o gram.go -v "" gram.y

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/cptaffe/automaton/lex"
	"github.com/cptaffe/automaton/seq"
	"github.com/cptaffe/automaton/seq/keypad/internal/scan"
)

const keys = "0123456789*#"

const Answered = "answered"

func Alphabet(g *seq.Grammar) *seq.Alphabet {
	var delays []string
	for name := range g.Delays {
		if name != Answered {
			delays = append(delays, name)
		}
	}
	sort.Strings(delays)
	return seq.NewAlphabet(strings.Split(keys, ""), append([]string{Answered}, delays...)...)
}

func classOf(r rune) ([]seq.Sym, bool) {
	from := strings.IndexRune(keys, r)
	switch r {
	case 'N':
		from = 2
	case 'X':
		from = 0
	default:
		if from < 0 {
			return nil, false
		}
		return []seq.Sym{seq.Sym(from + 1)}, true
	}
	var out []seq.Sym
	for k := from; k <= 9; k++ {
		out = append(out, seq.Sym(k+1))
	}
	return out, true
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
	text       string
	line       int
	start, end int
}

type symv struct {
	c          seq.Class
	line       int
	start, end int
}

type pat struct {
	elems      []seq.Class
	line       int
	start, end int
}

type lexer struct {
	*lex.Reader
	src     string
	tok     int
	tokText string
	tokLine int
	block   []string
	bare    bool
	g       *seq.Grammar
	hands   map[int]string
	cur     string
}

type rule struct {
	seq.Rule
	hands string
}

type Tree struct {
	Grammar *seq.Grammar
	hands   map[int]string
}

func Parse(r io.Reader, from string) (*Tree, error) {
	b, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	in, err := lex.Open(scan.Spec, strings.NewReader(string(b)), "start", from, scan.TokNewline)
	if err != nil {
		return nil, err
	}
	l := &lexer{Reader: in, src: string(b), bare: true, g: &seq.Grammar{Name: in.Name()}, hands: map[int]string{}}
	if yyParse(l) != 0 && l.Err() == nil {
		l.FailAt(l.tokLine, "%s", l.unexpected())
	}
	if err := l.Err(); err != nil {
		return nil, err
	}
	if err := seq.Check(l.g); err != nil {
		return nil, err
	}
	return &Tree{Grammar: l.g, hands: l.hands}, nil
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
		if tok.Kind == scan.TokComment {
			l.comment(tok.Text, l.bare)
			l.bare = false
			l.Skip()
			continue
		}
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
			v.word = word{tok.Text, tok.At.Line, tok.Start, tok.End}
		case scan.TokSym:
			v.sym = symv{symbolOf(tok.Text), tok.At.Line, tok.Start, tok.End}
			return SYM, tok.Text
		case scan.TokDeclaration:
			l.Fail("%s is not a declaration", tok.Text)
			return 0, ""
		case scan.TokName:
			v.word = word{tok.Text, tok.At.Line, tok.Start, tok.End}
			return IDENT, tok.Text
		}
		return kinds[tok.Kind], tok.Text
	}
}

func symbolOf(text string) seq.Class {
	syms, _ := classOf(rune(text[0]))
	return seq.Class{Syms: syms, Many: strings.HasSuffix(text, "+"), Text: text}
}

func one(s symv) pat {
	return pat{elems: []seq.Class{s.c}, line: s.line, start: s.start, end: s.end}
}

func add(p pat, s symv) pat {
	return pat{elems: append(p.elems, s.c), line: p.line, start: p.start, end: s.end}
}

func event(t word) symv {
	return symv{seq.Class{Event: t.text, Text: t.text}, t.line, t.start, t.end}
}

func (l *lexer) text(p pat) string { return l.src[p.start:p.end] }

func (l *lexer) unexpected() string {
	switch l.tok {
	case SYM:
		return fmt.Sprintf("%q is a pattern symbol, and one cannot stand here", l.tokText)
	case IDENT:
		return fmt.Sprintf("%q is a name, and one cannot stand here", l.tokText)
	}
	return fmt.Sprintf("%s cannot stand here", l.tokText)
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

func (l *lexer) rule(r rule) {
	label := l.says()
	if l.cur == "" {
		l.FailAt(r.Line, "%q is above every scope, where only a `...` rule can be", r.Pattern)
		return
	}
	if r.Action == "" && r.Goto == "" && r.hands == "" {
		l.FailAt(r.Line, "%q has a pattern and nowhere to go", r.Pattern)
		return
	}
	if err := named(r.Action); err != nil {
		l.FailAt(r.Line, "%s", err)
		return
	}
	r.Scope, r.Label = l.cur, label
	if r.hands != "" {
		l.hands[len(l.g.Rules)] = r.hands
	}
	l.g.Rules = append(l.g.Rules, r.Rule)
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
