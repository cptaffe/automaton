package gen

import "github.com/cptaffe/automaton"

type FlatActs struct {
	Emit string
	Do   string
	Goto string
	Push string
	Pop  bool
	Drop bool
}

type FlatRule struct {
	Expr    automaton.Expr[rune]
	Acts    FlatActs
	Include string
	Inlined bool
	Scope   string
	Line    int
	Text    string
}

type FlatScope struct {
	Name string
	Line int
}

type Flat struct {
	Defs     []Def
	Name     string
	Path     string
	Scopes   []FlatScope
	Rules    []FlatRule
	Shadowed []string
}

func (f *Flat) Scope(name string) *FlatScope {
	for i := range f.Scopes {
		if f.Scopes[i].Name == name {
			return &f.Scopes[i]
		}
	}

	return nil
}

func (f *Flat) Of(name string) []FlatRule {
	var out []FlatRule

	for _, r := range f.Rules {
		if r.Scope == name && r.Include == "" {
			out = append(out, r)
		}
	}

	return out
}
