//go:generate go tool goyacc -l -p yy -o gram.go -v "" gram.y

package gen

import (
	"path/filepath"
	"strings"

	"github.com/cptaffe/automaton/fa"
)

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
	Expr    fa.Expr[rune]
	Acts    Acts
	Include string
	Scope   string
	Line    int
	Text    string
}

type Scope struct {
	Name  string
	Param string
	Line  int
}

type Def struct {
	Name string
	Expr fa.Expr[rune]
}

type File struct {
	Defs []Def

	Path     string
	Scopes   []Scope
	Rules    []Rule
	Shadowed []string
}

func (f *File) Name() string { return base(f.Path) }

func base(path string) string {
	return strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
}

func (f *File) Scope(name string) *Scope {
	for i := range f.Scopes {
		if f.Scopes[i].Name == name {
			return &f.Scopes[i]
		}
	}

	return nil
}
