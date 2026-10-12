package gen

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/cptaffe/automaton"
	"github.com/cptaffe/automaton/lex"
)

func table(name string) (*unicode.RangeTable, bool) {
	for _, from := range []map[string]*unicode.RangeTable{
		unicode.Categories, unicode.Scripts, unicode.Properties,
	} {
		if t, ok := from[name]; ok {
			return t, true
		}
	}

	return nil, false
}

func predefined(bare bool) map[string]automaton.Expr[rune] {
	out := map[string]automaton.Expr[rune]{"any": automaton.Lit(automaton.Span(0, utf8.MaxRune))}

	if bare {
		return out
	}

	for _, d := range prelude() {
		out[d.Name] = d.Expr
	}

	return out
}

func given(bare bool) map[string]string {
	out := map[string]string{"any": "is built in"}

	if bare {
		return out
	}

	for _, d := range prelude() {
		out[d.Name] = "is in the prelude"
	}

	return out
}

func (p *parser) define(w word, e automaton.Expr[rune]) {
	name := w.text

	if was := p.given[name]; was != "" {
		p.file.Shadowed = append(p.file.Shadowed,
			fmt.Sprintf("%s:%d: %s is defined here and %s; the definition wins",
				p.From(), w.line, name, was))
		delete(p.given, name)
		p.named[name] = e
		p.order = append(p.order, name)
		p.file.Defs = append(p.file.Defs, Def{Name: name, Expr: e})

		return
	}

	if _, ok := unicodeClass(name); ok {
		p.file.Shadowed = append(p.file.Shadowed,
			fmt.Sprintf("%s:%d: %s is defined here and is a class name already; the definition wins",
				p.From(), w.line, name))
		p.named[name] = e
		p.order = append(p.order, name)
		p.file.Defs = append(p.file.Defs, Def{Name: name, Expr: e})

		return
	}

	if _, ok := p.named[name]; ok {
		p.FailAt(w.line, "%s is defined twice", name)

		return
	}

	p.named[name] = e
	p.order = append(p.order, name)
	p.file.Defs = append(p.file.Defs, Def{Name: name, Expr: e})
}

func (p *parser) use(w word) automaton.Expr[rune] {
	name := w.text

	if e, ok := p.named[name]; ok {
		return e
	}

	if s, ok := unicodeClass(name); ok {
		return automaton.Lit(s)
	}

	if also := spelling(name); also != "" {
		p.FailAt(w.line, "%s is written %s here", name, also)
	} else {
		p.FailAt(w.line, "%s is not defined yet; there is %s", name, known(p.bare))
	}

	return automaton.Empty[rune]()
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
		p.FailAt(p.Statement(), "a rule is above every scope")

		return
	}

	if r.Acts.Do != "" && r.Acts.Emit == "" {
		p.FailAt(p.Statement(), "do %s with nothing emitted: a do rides on a token", r.Acts.Do)

		return
	}

	if r.Include == "" && r.Acts.Emit == "" && r.Acts.Goto == "" && !r.Acts.Drop {
		p.FailAt(p.Statement(), "a rule that matches and does nothing")

		return
	}

	if r.Acts.Drop && r.Acts.Emit != "" {
		p.FailAt(p.Statement(), "a rule cannot both ignore what it reads and emit it")

		return
	}

	r.Scope, r.Line, r.Text = p.ctx, p.Statement(), p.Source(p.Statement())
	p.file.Rules = append(p.file.Rules, r)
}

func (p *parser) text(s string) automaton.Expr[rune] {
	var e []automaton.Expr[rune]

	for _, r := range s {
		e = append(e, automaton.Lit(automaton.One(r)))
	}

	if len(e) == 0 {
		p.FailAt(p.Line(), "an empty literal matches nothing, so it would never advance")

		return automaton.Empty[rune]()
	}

	return automaton.Cat(e...)
}

func (p *parser) without(a, b automaton.Expr[rune]) automaton.Expr[rune] {
	x, ok := a.Single()

	if !ok {
		p.FailAt(p.Line(), "the left of a - reads more than one rune, and a difference takes one set of runes from another")

		return a
	}

	y, ok := b.Single()

	if !ok {
		p.FailAt(p.Line(), "the right of a - reads more than one rune, and a difference takes one set of runes from another")

		return a
	}

	got := x.Without(y)

	if got.Empty() {
		p.FailAt(p.Line(), "what is left reads nothing at all")

		return a
	}

	return automaton.Lit(got)
}

func (p *parser) anycase(s string) automaton.Expr[rune] {
	var e []automaton.Expr[rune]

	for _, r := range s {
		e = append(e, automaton.Lit(automaton.Text(string(unicode.ToLower(r))+string(unicode.ToUpper(r)))))
	}

	if len(e) == 0 {
		p.FailAt(p.Line(), "an empty literal matches nothing, so it would never advance")

		return automaton.Empty[rune]()
	}

	return automaton.Cat(e...)
}

func (p *parser) runes(s string) automaton.Set[rune] { return automaton.Text(s) }

func (p *parser) span(lo, hi string) automaton.Set[rune] {
	a, b := []rune(lo), []rune(hi)

	if len(a) != 1 || len(b) != 1 {
		p.FailAt(p.Line(), "a range runs between one rune and one rune, not %q and %q", lo, hi)

		return automaton.Set[rune]{}
	}

	if a[0] > b[0] {
		p.FailAt(p.Line(), "the range %q to %q runs backwards", lo, hi)

		return automaton.Set[rune]{}
	}

	return automaton.Span(a[0], b[0])
}

func (p *parser) category(name string) automaton.Set[rune] {
	t, ok := table(name)

	if !ok {
		p.FailAt(p.Line(), "%s is not a unicode category, script or property", name)

		return automaton.Set[rune]{}
	}

	return automaton.Category(t)
}

func builtinNames(bare bool) []string {
	out := []string{"any"}

	if !bare {
		for _, d := range prelude() {
			out = append(out, d.Name)
		}
	}

	sort.Strings(out)

	return out
}

func known(bare bool) string {
	return strings.Join(append(builtinNames(bare), unicodeNames()...), ", ")
}

func Parse(r io.Reader, from string) (*File, error) { return parse(r, from, false) }

func parse(r io.Reader, from string, bare bool) (*File, error) {
	p, err := newParser(r, from, bare)

	if err != nil {
		return nil, err
	}

	if yyParse(p) != 0 && p.Err() == nil {
		p.FailAt(p.Line(), "%s", p.unexpected())
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

func (p *parser) negate(s automaton.Set[rune]) automaton.Set[rune] {
	return automaton.Span[rune](0, utf8.MaxRune).Without(s)
}
