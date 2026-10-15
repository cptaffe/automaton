package seq

import (
	"time"

	"github.com/cptaffe/automaton/fa"
)

type Sym int

type Alphabet struct {
	keys  int
	names []string
	by    map[string]Sym
}

func NewAlphabet(keys []string, events ...string) *Alphabet {
	a := &Alphabet{by: map[string]Sym{}}
	for _, k := range keys {
		a.add(k)
	}
	a.keys = len(a.names)
	for _, e := range events {
		a.add(e)
	}
	return a
}

func (a *Alphabet) add(name string) {
	if _, ok := a.by[name]; ok {
		return
	}
	a.by[name] = Sym(len(a.names))
	a.names = append(a.names, name)
}

func (a *Alphabet) Size() int          { return len(a.names) }
func (a *Alphabet) Keys() int          { return a.keys }
func (a *Alphabet) IsKey(s Sym) bool   { return s >= 0 && int(s) < a.keys }
func (a *Alphabet) IsEvent(s Sym) bool { return int(s) >= a.keys && int(s) < len(a.names) }
func (a *Alphabet) Of(name string) (Sym, bool) {
	s, ok := a.by[name]
	return s, ok
}
func (a *Alphabet) Name(s Sym) string {
	if s < 0 || int(s) >= len(a.names) {
		return ""
	}
	return a.names[s]
}

type Atom struct {
	Syms  []Sym
	Event string
	Any   bool
	Text  string

	sym  Sym
	keys fa.Set[Sym]
}

func (c Atom) IsEvent() bool { return c.Event != "" }

func (c Atom) set() fa.Set[Sym] {
	if c.Event != "" {
		return fa.One(c.sym)
	}
	if c.Any {
		return c.keys
	}
	rs := make([]fa.Range[Sym], 0, len(c.Syms))
	for _, s := range c.Syms {
		rs = append(rs, fa.Range[Sym]{Lo: s, Hi: s})
	}
	return fa.Of(rs...)
}

func (c Atom) wide() bool {
	return c.Any || len(c.Syms) > 1
}

type Rule struct {
	Scope   string
	Pattern string
	Expr    Expr
	Peek    Expr
	Action  string
	Goto    string
	Arg     string
	Include string
	When    Cond
	Label   string
	Line    int
}

type Any struct {
	Scope   string
	Pattern string
	Elem    Expr
	Action  string
	Goto    string
	Arg     string
	When    Cond
	Label   string
	Line    int
}

type Grammar struct {
	Name   string
	Note   string
	Scopes []string
	Params map[string]string
	Rules  []Rule
	Any    []Any
	Delays map[string]time.Duration
}
