package seq

import (
	"sort"
	"time"
)

type Sym int

type Alphabet struct {
	keys  int
	names []string
	by    map[string]Sym
}

func NewAlphabet(keys []string, events ...string) *Alphabet {
	a := &Alphabet{names: []string{""}, by: map[string]Sym{}}
	for _, k := range keys {
		a.add(k)
	}
	a.keys = len(a.names) - 1
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

func (a *Alphabet) Size() int          { return len(a.names) - 1 }
func (a *Alphabet) Keys() int          { return a.keys }
func (a *Alphabet) IsKey(s Sym) bool   { return s >= 1 && int(s) <= a.keys }
func (a *Alphabet) IsEvent(s Sym) bool { return int(s) > a.keys && int(s) < len(a.names) }
func (a *Alphabet) Of(name string) Sym { return a.by[name] }
func (a *Alphabet) Name(s Sym) string {
	if s < 1 || int(s) >= len(a.names) {
		return ""
	}
	return a.names[s]
}

type Class struct {
	Syms  []Sym
	Event string
	Many  bool
	Text  string

	sym Sym
}

func (c Class) IsEvent() bool { return c.Event != "" }

func (c Class) has(s Sym) bool {
	if c.Event != "" {
		return c.sym == s
	}
	i := sort.Search(len(c.Syms), func(i int) bool { return c.Syms[i] >= s })
	return i < len(c.Syms) && c.Syms[i] == s
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
	Hands   string
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
	When    Cond
	Label   string
	Line    int

	key Sym
}

type Grammar struct {
	Name    string
	Note    string
	Scopes  []string
	Rules   []Rule
	Any     []Any
	Delays  map[string]time.Duration
	Settles map[string]string
}
