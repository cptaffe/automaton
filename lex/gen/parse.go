package gen

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/cptaffe/automaton"
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

func predefined() map[string]Expr {
	out := map[string]Expr{"any": Lit(automaton.Span(0, utf8.MaxRune))}

	if bare {
		return out
	}

	for _, d := range prelude() {
		out[d.Name] = d.Expr
	}

	return out
}

func given() map[string]string {
	out := map[string]string{"any": "is built in"}

	if bare {
		return out
	}

	for _, d := range prelude() {
		out[d.Name] = "is in the prelude"
	}

	return out
}

func (p *parser) define(name string, e Expr) {

	if was := p.given[name]; was != "" {
		p.file.Shadowed = append(p.file.Shadowed,
			fmt.Sprintf("%s:%d: %s is defined here and %s; the definition wins",
				p.From(), p.declLine, name, was))
		delete(p.given, name)
		p.named[name] = e
		p.order = append(p.order, name)
		p.file.Defs = append(p.file.Defs, Def{Name: name, Expr: e})

		return
	}

	if _, ok := unicodeClass(name); ok {
		p.file.Shadowed = append(p.file.Shadowed,
			fmt.Sprintf("%s:%d: %s is defined here and is a class name already; the definition wins",
				p.From(), p.declLine, name))
		p.named[name] = e
		p.order = append(p.order, name)
		p.file.Defs = append(p.file.Defs, Def{Name: name, Expr: e})

		return
	}

	if _, ok := p.named[name]; ok {
		p.FailAt(p.declLine, "%s is defined twice", name)

		return
	}

	p.named[name] = e
	p.order = append(p.order, name)
	p.file.Defs = append(p.file.Defs, Def{Name: name, Expr: e})
}

func (p *parser) use(name string) Expr {

	if e, ok := p.named[name]; ok {
		return e
	}

	if s, ok := unicodeClass(name); ok {
		return Lit(s)
	}

	if also := spelling(name); also != "" {
		p.FailAt(p.Line(), "%s is written %s here", name, also)
	} else {
		p.FailAt(p.Line(), "%s is not defined yet; there is %s", name, known())
	}

	return automaton.Empty()
}

func (p *parser) scope(name, param, includes string) {
	if p.file.Scope(name) != nil {
		p.FailAt(p.declLine, "%s is opened twice", name)

		return
	}

	if param == "this" {
		p.FailAt(p.declLine, "a scope cannot take a parameter called this, because this is the scope that used it")

		return
	}

	p.ctx = name
	p.file.Scopes = append(p.file.Scopes,
		Scope{Name: name, Param: param, Includes: includes, Line: p.declLine})
}

func (p *parser) rule(r Rule) {
	if p.ctx == "" {
		p.FailAt(p.Line(), "a rule is above every scope")

		return
	}

	if r.Acts.Emit == "" && r.Acts.Do == "" && r.Acts.Goto == "" && !r.Acts.Drop {
		p.FailAt(p.Line(), "a rule that matches and does nothing")

		return
	}

	if r.Acts.Drop && r.Acts.Emit != "" {
		p.FailAt(p.Line(), "a rule cannot both ignore what it reads and emit it")

		return
	}

	r.Scope, r.Line, r.Text = p.ctx, p.Statement(), p.Source(p.Statement())
	p.file.Rules = append(p.file.Rules, r)
}

func (p *parser) text(s string) Expr {
	var e []Expr

	for _, r := range s {
		e = append(e, Lit(automaton.One(r)))
	}

	if len(e) == 0 {
		p.FailAt(p.Line(), "an empty literal matches nothing, so it would never advance")

		return automaton.Empty()
	}

	return Cat(e...)
}

func (p *parser) without(a, b Expr) Expr {
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

	return Lit(got)
}

func (p *parser) anycase(s string) Expr {
	var e []Expr

	for _, r := range s {
		e = append(e, Lit(automaton.Text(string(unicode.ToLower(r))+string(unicode.ToUpper(r)))))
	}

	if len(e) == 0 {
		p.FailAt(p.Line(), "an empty literal matches nothing, so it would never advance")

		return automaton.Empty()
	}

	return Cat(e...)
}

func (p *parser) runes(s string) Set { return automaton.Text(s) }

func (p *parser) span(lo, hi string) Set {
	a, b := []rune(lo), []rune(hi)

	if len(a) != 1 || len(b) != 1 {
		p.FailAt(p.Line(), "a range runs between one rune and one rune, not %q and %q", lo, hi)

		return Set{}
	}

	if a[0] > b[0] {
		p.FailAt(p.Line(), "the range %q to %q runs backwards", lo, hi)

		return Set{}
	}

	return automaton.Span(a[0], b[0])
}

func (p *parser) category(name string) Set {
	t, ok := table(name)

	if !ok {
		p.FailAt(p.Line(), "%s is not a unicode category, script or property", name)

		return Set{}
	}

	return automaton.Category(t)
}

func builtinNames() []string {
	out := []string{"any"}

	if !bare {
		for _, d := range prelude() {
			out = append(out, d.Name)
		}
	}

	sort.Strings(out)

	return out
}

func isBuiltin(name string) bool { return name == "any" }

func known() string {
	return strings.Join(append(builtinNames(), unicodeNames()...), ", ")
}

func Parse(r io.Reader, from string) (*File, error) {
	p, err := newParser(r, from)

	if err != nil {
		return nil, err
	}

	if yyParse(p) != 0 && p.Err() == nil {
		p.FailAt(p.Line(), "%s", p.unexpected())
	}

	if err := p.Err(); err != nil {
		return nil, err
	}

	if len(p.file.Scopes) == 0 {

		if !bare {
			return nil, fmt.Errorf("%s: no scopes, so there is nowhere to begin", from)
		}
	}

	for _, s := range p.file.Scopes {
		if s.Includes != "" && p.file.Scope(s.Includes) == nil {
			return nil, fmt.Errorf("%s:%d: %s includes %s, and there is no such scope",
				from, s.Line, s.Name, s.Includes)
		}
	}

	return p.file, nil
}
