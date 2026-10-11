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
	Expr    automaton.Expr
	Acts    FlatActs
	Include string
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

	var walk func(at string, seen map[string]bool)

	walk = func(at string, seen map[string]bool) {
		if seen[at] {
			return
		}

		seen[at] = true

		for _, r := range f.Rules {
			switch {
			case r.Scope != at:
			case r.Include != "":
				walk(r.Include, seen)
			default:
				out = append(out, r)
			}
		}

		delete(seen, at)
	}

	walk(name, map[string]bool{})

	return out
}
