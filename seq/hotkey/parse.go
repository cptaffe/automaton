package hotkey

//go:generate go run ../../cmd/genlex -go internal/scan/scan.gen.go -package scan hotkey.lex
//go:generate go tool goyacc -l -p yy -o gram.go -v "" gram.y

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/cptaffe/automaton/lex"
	"github.com/cptaffe/automaton/seq"
	"github.com/cptaffe/automaton/seq/hotkey/internal/scan"
)

var modifiers = []string{"command", "shift", "control", "option"}

type Rules struct {
	Grammar *seq.Grammar
	Table   *seq.Table
	Prelude string
}

type word struct {
	text string
	line int
}

type pat struct {
	words []word
	line  int
	ahead int
}

type rule struct {
	keys   pat
	action string
	block  bool
	to     string
	at     int
}

type lexer struct {
	*lex.Reader
	tok     int
	text    string
	line    int
	bare    bool
	labels  map[int]string
	g       *seq.Grammar
	cur     string
	rules   []rule
	anys    []rule
	scopes  map[string]bool
	prelude string
}

var kinds = map[int]int{
	scan.TokNewline: NEWLINE,
	scan.TokColon:   COLON,
	scan.TokArrow:   ARROW,
	scan.TokWhen:    WHEN,
	scan.TokPeek:    PEEK,
	scan.TokEq:      EQ,
	scan.TokNe:      NE,
	scan.TokMatch:   MATCH,
	scan.TokAnd:     AND,
	scan.TokOr:      OR,
	scan.TokNot:     NOT,
	scan.TokLparen:  LPAREN,
	scan.TokRparen:  RPAREN,
	scan.TokText:    TEXT,
	scan.TokWord:    WORD,
	scan.TokString:  STRING,
	scan.TokIdent:   IDENT,
	scan.TokDelay:   DELAY,
	scan.TokDots:    DOTS,
	scan.TokLbrace:  LBRACE,
	scan.TokRbrace:  RBRACE,
	scan.TokOpen:    OPEN,
	scan.TokClose:   CLOSE,
}

func Parse(r io.Reader, from string, kb Keyboard) (*Rules, error) {
	if kb == nil {
		kb = US
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	in, err := lex.Open(ctx, scan.Spec, r, "start", from)
	if err != nil {
		return nil, err
	}
	l := &lexer{Reader: in, bare: true, labels: map[int]string{}, scopes: map[string]bool{},
		g: &seq.Grammar{Name: in.Name()}}
	failed := yyParse(l) != 0
	switch {
	case l.Err() != nil:
	case failed && l.tok != 0:
		l.FailAt(l.line, "%s cannot stand here", strings.TrimSpace(l.text))
	case failed:
		l.FailAt(l.line, "the file ends in the middle of a rule")
	}
	if err := l.Err(); err != nil {
		return nil, err
	}
	a, err := l.resolve(kb)
	if err != nil {
		return nil, err
	}
	t, err := seq.Compile(l.g, a)
	if err != nil {
		return nil, err
	}
	if len(t.Conflicts) > 0 {
		return nil, errors.New(strings.Join(t.Conflicts, "\n"))
	}
	return &Rules{Grammar: l.g, Table: t, Prelude: l.prelude}, nil
}

func (l *lexer) Lex(v *yySymType) int {
	for {
		tok, ok := l.Next()
		if !ok {
			return 0
		}
		l.text, l.line = tok.Text, tok.At.Line
		switch tok.Kind {
		case scan.TokComment:
			if !l.bare {
				l.labels[tok.At.Line] = strings.TrimSpace(strings.TrimPrefix(tok.Text, "#"))
			}
			continue
		case scan.TokNewline:
			l.bare = true
		default:
			l.bare = false
		}
		text := tok.Text
		if tok.Kind == scan.TokString {
			text = text[1 : len(text)-1]
		}
		v.word, v.str = word{text, tok.At.Line}, text
		l.tok = kinds[tok.Kind]
		return l.tok
	}
}

func (l *lexer) Error(string) {}

func (l *lexer) addPrelude(text string) {
	if l.prelude != "" {
		l.prelude += "\n"
	}
	l.prelude += strings.TrimRight(text, "\n")
}

func (l *lexer) scope(name word) {
	l.cur = name.text
	if l.scopes[name.text] {
		l.FailAt(name.line, "scope %s is declared twice", name.text)
		return
	}
	l.scopes[name.text] = true
	l.g.Scopes = append(l.g.Scopes, name.text)
}

func (l *lexer) include(to word, when seq.Cond) {
	if l.cur == "" {
		l.FailAt(to.line, "-> %s is above every scope, and it reads %s's rules into one", to.text, to.text)
		return
	}
	l.g.Rules = append(l.g.Rules, seq.Rule{Scope: l.cur, Include: to.text, When: when, Line: to.line})
}

func (l *lexer) delay(line int, name, of string) {
	if name == "any" {
		l.FailAt(line, "%%delay any: any is every key, not a delay")
		return
	}
	if l.cur != "" {
		l.FailAt(line, "%%delay %s belongs above every scope", name)
		return
	}
	d, err := time.ParseDuration(of)
	if err != nil || d <= 0 {
		l.FailAt(line, "%%delay %s %s: %s is not a duration", name, of, of)
		return
	}
	if _, ok := l.g.Delays[name]; ok {
		l.FailAt(line, "%%delay %s is declared twice", name)
		return
	}
	if l.g.Delays == nil {
		l.g.Delays = map[string]time.Duration{}
	}
	l.g.Delays[name] = d
}

func (l *lexer) rule(r rule) {
	if l.cur == "" {
		l.FailAt(r.keys.line, "a rule above every scope; only a ... rule can be there")
		return
	}
	if !r.block && r.to == "" {
		l.FailAt(r.keys.line, "a rule with keys and nowhere to go")
		return
	}
	r.at = len(l.g.Rules)
	l.rules = append(l.rules, r)
	l.g.Rules = append(l.g.Rules, seq.Rule{Scope: l.cur, Action: r.action, Goto: r.to, Label: l.labels[r.keys.line], Line: r.keys.line})
}

func (l *lexer) anyRule(r rule) {
	if len(r.keys.words) != 1 || r.keys.ahead > 0 {
		l.FailAt(r.keys.line, "... takes one key")
		return
	}
	l.anys = append(l.anys, r)
	l.g.Any = append(l.g.Any, seq.Any{Scope: l.cur, Action: r.action, Goto: r.to, Label: l.labels[r.keys.line], Line: r.keys.line})
}

func (l *lexer) compare(name word, op seq.Op, value word) seq.Cond {
	c, err := seq.NewCompare(name.text, op, value.text)
	if err != nil {
		l.FailAt(name.line, "%v", err)
		return seq.Name(name.text)
	}
	return c
}

type resolver struct {
	g    *seq.Grammar
	kb   Keyboard
	a    *seq.Alphabet
	errs []string
}

func (l *lexer) resolve(kb Keyboard) (*seq.Alphabet, error) {
	r := &resolver{g: l.g, kb: kb}
	for _, x := range append(append([]rule(nil), l.rules...), l.anys...) {
		for _, w := range x.keys.words {
			r.key(w)
		}
	}
	if len(r.errs) > 0 {
		return nil, fmt.Errorf("%s", strings.Join(r.errs, "\n"))
	}
	r.alphabet()
	for _, x := range l.rules {
		g := &l.g.Rules[x.at]
		g.Elems = r.classes(x.keys)
		g.Ahead = x.keys.ahead
		g.Pattern = x.keys.pattern()
	}
	for i, x := range l.anys {
		l.g.Any[i].Elem = r.class(x.keys.words[0])
		l.g.Any[i].Pattern = x.keys.words[0].text
	}
	return r.a, nil
}

func (r *resolver) fail(line int, f string, a ...any) {
	r.errs = append(r.errs, fmt.Sprintf("%s:%d: %s", r.g.Name, line, fmt.Sprintf(f, a...)))
}

func (r *resolver) key(w word) (string, bool) {
	if w.text == "any" {
		return "", false
	}
	if _, ok := r.g.Delays[w.text]; ok {
		return w.text, true
	}
	k, err := Canon(w.text)
	if err != nil {
		r.fail(w.line, "%s: %v", w.text, err)
		return "", false
	}
	if !r.kb.Has(Name(k)) {
		r.fail(w.line, "%s: no such key", w.text)
		return "", false
	}
	return k, false
}

func (r *resolver) alphabet() {
	var events []string
	for name := range r.g.Delays {
		events = append(events, name)
	}
	sort.Strings(events)
	r.a = seq.NewAlphabet(r.kb.chords(), events...)
}

func (r *resolver) class(w word) seq.Class {
	if w.text == "any" {
		return seq.Class{Any: true, Text: w.text}
	}
	k, event := r.key(w)
	if event {
		return seq.Class{Event: k, Text: w.text}
	}
	s, _ := r.a.Of(k)
	return seq.Class{Syms: []seq.Sym{s}, Text: w.text}
}

func (r *resolver) classes(p pat) []seq.Class {
	var out []seq.Class
	for _, w := range p.words {
		out = append(out, r.class(w))
	}
	return out
}

func (p pat) pattern() string {
	var texts []string
	for i, w := range p.words {
		if i == len(p.words)-p.ahead {
			texts = append(texts, "peek")
		}
		texts = append(texts, w.text)
	}
	return strings.Join(texts, " ")
}

func Canon(s string) (string, error) {
	mods := map[string]bool{}
	for {
		found := false
		for _, m := range modifiers {
			if p := m + "-"; strings.HasPrefix(s, p) && len(s) > len(p) {
				if mods[m] {
					return "", fmt.Errorf("%s twice", m)
				}
				mods[m] = true
				s = s[len(p):]
				found = true
			}
		}
		if !found {
			break
		}
	}
	var have []string
	for m := range mods {
		have = append(have, m)
	}
	return Key(have, s), nil
}

func Key(mods []string, name string) string {
	have := map[string]bool{}
	for _, m := range mods {
		have[m] = true
	}
	var parts []string
	for _, m := range modifiers {
		if have[m] {
			parts = append(parts, m)
		}
	}
	return strings.Join(append(parts, name), "-")
}

func Name(key string) string {
	for {
		cut := false
		for _, m := range modifiers {
			if p := m + "-"; strings.HasPrefix(key, p) && len(key) > len(p) {
				key, cut = key[len(p):], true
			}
		}
		if !cut {
			return key
		}
	}
}
