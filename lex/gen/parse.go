package gen

import (
	"context"
	"fmt"
	"io"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/cptaffe/automaton/fa"
	"github.com/cptaffe/automaton/lex"
)

func (p *parser) preload() {
	pre := prelude()
	names := make([]string, 0, len(pre)+1)
	names = append(names, "any")

	for _, d := range pre {
		p.named[d.Name] = d.Expr
		p.given[d.Name] = "is in the prelude"
		names = append(names, d.Name)
	}

	sort.Strings(names)

	for _, name := range names {
		p.file.Defs = append(p.file.Defs, Def{Name: name, Expr: p.named[name]})
	}
}

func (p *parser) define(w word, e fa.Expr[rune]) {
	name := w.text
	was := p.given[name]
	_, class := unicodeClass(name)
	_, taken := p.named[name]

	switch {
	case was != "":
		p.file.Shadowed = append(p.file.Shadowed,
			fmt.Sprintf("%s:%d: %s is defined here and %s; the definition wins",
				p.From(), w.line, name, was))
		delete(p.given, name)
	case class:
		p.file.Shadowed = append(p.file.Shadowed,
			fmt.Sprintf("%s:%d: %s is defined here and is a class name already; the definition wins",
				p.From(), w.line, name))
	case taken:
		p.FailAt(w.line, "%s is defined twice", name)

		return
	}

	p.named[name] = e
	p.file.Defs = append(p.file.Defs, Def{Name: name, Expr: e})
}

func (p *parser) use(w word) fa.Expr[rune] {
	name := w.text

	if e, ok := p.named[name]; ok {
		return e
	}

	if s, ok := unicodeClass(name); ok {
		return fa.Lit(s)
	}

	if also := spelling(name); also != "" {
		p.FailAt(w.line, "%s is written %s here", name, also)
	} else {
		p.FailAt(w.line, "%s is not defined yet; there is %s", name, p.known())
	}

	return fa.Empty[rune]()
}

func (p *parser) scope(w word, param string) {
	if p.file.Scope(w.text) != nil {
		p.FailAt(w.line, "%s is opened twice", w.text)

		return
	}

	if param == "this" {
		p.FailAt(w.line, "a scope cannot take a parameter called this, because this is the scope that used it")

		return
	}

	p.ctx = w.text
	p.file.Scopes = append(p.file.Scopes, Scope{Name: w.text, Param: param, Line: w.line})
}

func (p *parser) rule(r Rule) {
	if p.ctx == "" {
		p.FailAt(p.stmt, "a rule is above every scope")

		return
	}

	if r.Acts.Do != "" && r.Acts.Emit == "" {
		p.FailAt(p.stmt, "do %s with nothing emitted: a do rides on a token", r.Acts.Do)

		return
	}

	if r.Include == "" && r.Acts.Emit == "" && r.Acts.Goto == "" && !r.Acts.Drop {
		p.FailAt(p.stmt, "a rule that matches and does nothing")

		return
	}

	if r.Acts.Drop && r.Acts.Emit != "" {
		p.FailAt(p.stmt, "a rule cannot both ignore what it reads and emit it")

		return
	}

	r.Scope, r.Line, r.Text = p.ctx, p.stmt, p.stmtText
	p.file.Rules = append(p.file.Rules, r)
}

func (p *parser) text(s string) fa.Expr[rune] {
	e := make([]fa.Expr[rune], 0, len(s))

	for _, r := range s {
		e = append(e, fa.Lit(fa.One(r)))
	}

	if len(e) == 0 {
		p.FailAt(p.Line(), "an empty literal matches nothing, so it would never advance")

		return fa.Empty[rune]()
	}

	return fa.Cat(e...)
}

func (p *parser) without(a, b fa.Expr[rune]) fa.Expr[rune] {
	x, ok := a.Single()

	if !ok {
		p.FailAt(p.Line(), "the left of a \\ reads more than one rune, and a difference takes one set of runes from another")

		return a
	}

	y, ok := b.Single()

	if !ok {
		p.FailAt(p.Line(), "the right of a \\ reads more than one rune, and a difference takes one set of runes from another")

		return a
	}

	got := x.Without(y)

	if got.Empty() {
		p.FailAt(p.Line(), "what is left reads nothing at all")

		return a
	}

	return fa.Lit(got)
}

func (p *parser) anycase(s string) fa.Expr[rune] {
	e := make([]fa.Expr[rune], 0, len(s))

	for _, r := range s {
		e = append(e, fa.Lit(fa.Text(string(unicode.ToLower(r))+string(unicode.ToUpper(r)))))
	}

	if len(e) == 0 {
		p.FailAt(p.Line(), "an empty literal matches nothing, so it would never advance")

		return fa.Empty[rune]()
	}

	return fa.Cat(e...)
}

func (p *parser) runes(s string) fa.Set[rune] { return fa.Text(s) }

func (p *parser) span(lo, hi string) fa.Set[rune] {
	a, b := []rune(lo), []rune(hi)

	if len(a) != 1 || len(b) != 1 {
		p.FailAt(p.Line(), "a range runs between one rune and one rune, not %q and %q", lo, hi)

		return fa.Set[rune]{}
	}

	if a[0] > b[0] {
		p.FailAt(p.Line(), "the range %q to %q runs backwards", lo, hi)

		return fa.Set[rune]{}
	}

	return fa.Span(a[0], b[0])
}

func (p *parser) category(name string) fa.Set[rune] {
	s, ok := p.use(word{name, p.Line()}).Single()

	if !ok {
		p.FailAt(p.Line(), "[:%s:] is not a set of runes", name)

		return fa.Set[rune]{}
	}

	return s
}

func (p *parser) known() string {
	names := make([]string, 0, len(p.file.Defs))
	seen := map[string]bool{}

	for _, d := range p.file.Defs {
		if !seen[d.Name] {
			seen[d.Name] = true
			names = append(names, d.Name)
		}
	}

	return strings.Join(append(names, "or a unicode.* class"), ", ")
}

func Parse(r io.Reader, from string) (*File, error) { return parse(r, from, false) }

func parse(r io.Reader, from string, bare bool) (*File, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	p, err := newParser(ctx, r, from, bare)

	if err != nil {
		return nil, err
	}

	if yyParse(p) != 0 && p.Err() == nil {
		p.FailAt(p.blame(), "%s", p.unexpected())
	}

	if err := p.Err(); err != nil {
		return nil, err
	}

	if len(p.file.Scopes) == 0 && !bare {
		return nil, fmt.Errorf("%s: no scopes, so there is nowhere to begin", from)
	}

	for _, r := range p.file.Rules {
		if r.Include == "" {
			continue
		}

		s := p.file.Scope(r.Include)

		switch {
		case s == nil:
			return nil, &lex.Error{File: from, Line: r.Line, Err: fmt.Errorf("-> %s, and there is no such scope", r.Include)}
		case s.Param != "":
			return nil, &lex.Error{File: from, Line: r.Line, Err: fmt.Errorf("-> %s reads %s's rules here, and %s takes a parameter", r.Include, r.Include, r.Include)}
		}
	}

	return p.file, nil
}

func (p *parser) negate(s fa.Set[rune]) fa.Set[rune] {
	return fa.Span[rune](0, utf8.MaxRune).Without(s)
}
