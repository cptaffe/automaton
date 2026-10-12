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
	a.names = append(a.names, name)
	a.by[name] = Sym(len(a.names))
}

func (a *Alphabet) Size() int          { return len(a.names) }
func (a *Alphabet) Keys() int          { return a.keys }
func (a *Alphabet) IsKey(s Sym) bool   { return s >= 1 && int(s) <= a.keys }
func (a *Alphabet) IsEvent(s Sym) bool { return int(s) > a.keys && int(s) <= len(a.names) }
func (a *Alphabet) Of(name string) Sym { return a.by[name] }
func (a *Alphabet) Name(s Sym) string {
	if s < 1 || int(s) > len(a.names) {
		return ""
	}
	return a.names[s-1]
}

type Class struct {
	Syms  []Sym
	Event string
	Many  bool
	Text  string

	sym Sym
}

func (c Class) IsEvent() bool { return c.Event != "" }

func (c Class) set() fa.Set[Sym] {
	if c.Event != "" {
		return fa.One(c.sym)
	}
	var rs []fa.Range[Sym]
	for _, s := range c.Syms {
		rs = append(rs, fa.Range[Sym]{Lo: s, Hi: s})
	}
	return fa.Of(rs...)
}

func (c Class) wide() bool {
	return len(c.Syms) > 1
}

func (c Class) only() (Sym, bool) {
	if c.Event != "" {
		return c.sym, !c.Many
	}
	if len(c.Syms) == 1 && !c.Many {
		return c.Syms[0], true
	}
	return 0, false
}

type Rule struct {
	Scope   string
	Pattern string
	Elems   []Class
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
	Elem    Class
	Action  string
	Goto    string
	Arg     string
	When    Cond
	Label   string
	Line    int

	key Sym
}

type Grammar struct {
	Name    string
	Note    string
	Scopes  []string
	Params  map[string]string
	Rules   []Rule
	Any     []Any
	Delays  map[string]time.Duration
	Settles map[string]string
}
