package keypad

//go:generate go run ../../cmd/genlex -go internal/scan/scan.gen.go -package scan tree.lex
//go:generate go tool goyacc -l -p yy -o gram.go -v "" gram.y

import (
	"context"
	"fmt"
	"io"
	"slices"
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
		return []seq.Sym{seq.Sym(from)}, true
	}
	var out []seq.Sym
	for k := from; k <= 9; k++ {
		out = append(out, seq.Sym(k))
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
	scan.TokDelay:   DELAY,
	scan.TokDo:      DO,
	scan.TokWhen:    WHEN,
	scan.TokAnd:     AND,
	scan.TokOr:      OR,
	scan.TokNot:     NOT,
	scan.TokSym:     SYM,
	scan.TokName:    IDENT,
	scan.TokBar:     BAR,
	scan.TokPlus:    PLUS,
	scan.TokOpt:     OPT,
	scan.TokMinus:   MINUS,
}

type word struct {
	text string
	line int
	lead string
}

type symv struct {
	e    seq.Atom
	text string
	line int
	lead string
}

type pat struct {
	e    seq.Expr
	set  bool
	line int
	text string
}

type lexer struct {
	*lex.Reader
	tok     int
	tokText string
	tokLine int
	block   []string
	bare    bool
	end     int
	g       *seq.Grammar
	cur     string
	ahead   []lex.Token
	call    bool
}

type move struct{ to, arg string }

func Parse(r io.Reader, from string) (*seq.Grammar, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	in, err := lex.Open(ctx, scan.Spec, r, "start", from)
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
	return l.g, nil
}

func (l *lexer) Lex(v *yySymType) int {
	t, text := l.lex(v)
	l.tok, l.tokText = t, text
	return t
}

func (l *lexer) Error(string) {}

func (l *lexer) lex(v *yySymType) (int, string) {
	tok, lead, ok := l.token()
	if !ok {
		return 0, ""
	}
	v.word = word{tok.Text, tok.At.Line, lead}
	switch tok.Kind {
	case scan.TokNewline:
		if l.bare {
			l.block = nil
		}
		l.bare = true
		return NEWLINE, "a new line"
	case scan.TokSym:
		v.sym = symv{symbolOf(tok.Text), tok.Text, tok.At.Line, lead}
	case scan.TokDeclaration:
		l.Fail("%s is not a declaration", tok.Text)
		return 0, ""
	case scan.TokLparen:
		if l.call {
			l.call = false
			return CALL, tok.Text
		}
	}
	return kinds[tok.Kind], tok.Text
}

func (l *lexer) token() (lex.Token, string, bool) {
	for {
		tok, ok := l.pull()
		if !ok {
			return tok, "", false
		}
		l.tokLine = tok.At.Line
		if tok.Kind == scan.TokName && (l.tok == 0 || l.tok == NEWLINE) {
			l.call = l.header()
		}
		lead := ""
		if tok.At.Offset > l.end {
			lead = " "
		}
		l.end = tok.End()
		if tok.Kind != scan.TokComment {
			if tok.Kind != scan.TokNewline {
				l.bare = false
			}
			return tok, lead, true
		}
		l.comment(tok.Text, l.bare)
		l.bare = false
	}
}

func symbolOf(text string) seq.Atom {
	syms, _ := classOf(rune(text[0]))
	return seq.Atom{Syms: syms, Text: text}
}

func name(w word) pat {
	c := seq.Atom{Event: w.text, Text: w.text}
	if w.text == "any" {
		c = seq.Atom{Any: true, Text: w.text}
	}
	return pat{e: c, set: true, line: w.line, text: w.lead + w.text}
}

func cat(a, b pat) pat {
	items, ok := a.e.(seq.Cat)
	if !ok {
		items = seq.Cat{a.e}
	}
	return pat{e: append(items, b.e), line: a.line, text: a.text + b.text}
}

func alt(a pat, op word, b pat) pat {
	items, ok := a.e.(seq.Alt)
	if !ok {
		items = seq.Alt{a.e}
	}
	return pat{e: append(items, b.e), set: a.set && b.set, line: a.line, text: a.text + op.lead + "|" + b.text}
}

func repeat(a pat, op word, e seq.Expr) pat {
	return pat{e: e, line: a.line, text: a.text + op.lead + op.text}
}

func minus(a pat, op word, b pat) pat {
	return pat{e: seq.Minus{From: a.e, Take: b.e}, set: a.set && b.set, line: a.line, text: a.text + op.lead + "-" + b.text}
}

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
	if name == "any" {
		l.FailAt(line, "%%delay any: any is every key, not a delay")
		return
	}
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

func (l *lexer) scope(name word, param string) {
	l.says()
	l.cur = name.text
	if was, ok := l.g.Params[name.text]; ok && was != param || !ok && param != "" && slices.Contains(l.g.Scopes, name.text) {
		l.FailAt(name.line, "%s is opened again with another parameter", name.text)
		return
	}
	if param != "" {
		if l.g.Params == nil {
			l.g.Params = map[string]string{}
		}
		l.g.Params[name.text] = param
	}
	if !slices.Contains(l.g.Scopes, name.text) {
		l.g.Scopes = append(l.g.Scopes, name.text)
	}
}

func (l *lexer) rule(r seq.Rule) {
	label := l.says()
	if l.cur == "" {
		l.FailAt(r.Line, "%q is above every scope, where only a `...` rule can be", r.Pattern)
		return
	}
	if r.Action == "" && r.Goto == "" {
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

func (l *lexer) pull() (lex.Token, bool) {
	if len(l.ahead) > 0 {
		tok := l.ahead[0]
		l.ahead = l.ahead[1:]
		return tok, true
	}
	return l.Next()
}

func (l *lexer) header() bool {
	want := []int{scan.TokLparen, scan.TokName, scan.TokRparen, scan.TokColon}
	for len(l.ahead) < len(want) {
		tok, ok := l.Next()
		if !ok {
			return false
		}
		l.ahead = append(l.ahead, tok)
	}
	for i, k := range want {
		if l.ahead[i].Kind != k {
			return false
		}
	}
	return true
}
