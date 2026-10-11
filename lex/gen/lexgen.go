//go:generate go tool goyacc -l -p yy -o gram.go -v /dev/null gram.y

package gen

import "github.com/cptaffe/automaton"

type Set = automaton.Set

type Expr = automaton.Expr

func Lit(s Set) Expr     { return automaton.Lit(s) }
func Cat(e ...Expr) Expr { return automaton.Cat(e...) }
func Alt(e ...Expr) Expr { return automaton.Alt(e...) }
func Star(e Expr) Expr   { return automaton.Star(e) }
func Plus(e Expr) Expr   { return automaton.Plus(e) }
func Opt(e Expr) Expr    { return automaton.Opt(e) }

type Acts struct {
	Emit string
	Do   string
	Goto string
	Arg  string
	Drop bool
}

func (a Acts) and(b Acts) Acts {
	if b.Emit != "" {
		a.Emit = b.Emit
	}

	if b.Do != "" {
		a.Do = b.Do
	}

	if b.Goto != "" {
		a.Goto = b.Goto
	}

	if b.Arg != "" {
		a.Arg = b.Arg
	}

	if b.Drop {
		a.Drop = true
	}

	return a
}

type Rule struct {
	Expr  Expr
	Acts  Acts
	Scope string
	Line  int
	Text  string
}

type Scope struct {
	Name     string
	Param    string
	Includes string
	Line     int
}

type Def struct {
	Name string
	Expr Expr
}

type File struct {
	Defs []Def

	Name     string
	Scopes   []Scope
	Rules    []Rule
	Shadowed []string
}

func (f *File) Scope(name string) *Scope {
	for i := range f.Scopes {
		if f.Scopes[i].Name == name {
			return &f.Scopes[i]
		}
	}

	return nil
}
