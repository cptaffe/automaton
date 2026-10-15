package gen

import "github.com/cptaffe/automaton/fa"

type FlatActs struct {
	Emit string
	Do   string
	Goto string
	Push string
	Pop  bool
	Drop bool
}

type FlatRule struct {
	Expr    fa.Expr[rune]
	Acts    FlatActs
	Include string
	Inlined bool
	Scope   string
	Line    int
	Text    string
}

type FlatScope struct {
	Name string
	Of   string
	Line int
}

type Flat struct {
	Defs     []Def
	Path     string
	Scopes   []FlatScope
	Rules    []FlatRule
	Shadowed []string
}

func (f *Flat) Name() string { return base(f.Path) }

func (f *Flat) Of(name string) []FlatRule {
	var out []FlatRule

	for _, r := range f.Rules {
		if r.Scope == name && r.Include == "" {
			out = append(out, r)
		}
	}

	return out
}
