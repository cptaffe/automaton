package seq

import (
	"fmt"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/cptaffe/automaton/fa"
)

type Slice struct {
	From int
	To   int
}

type Accept struct {
	Action  string
	Goto    int
	When    Cond
	Pattern string
	Line    int
	Operand *Slice
}

type Scope struct {
	Name  string
	Start int
}

type Table struct {
	Name      string
	Note      string
	Alpha     *Alphabet
	Scopes    []Scope
	Next      [][]int
	Accept    [][]*Accept
	Live      [][]Cond
	Settle    []Sym
	Any       [][][]*Accept
	Delays    map[string]time.Duration
	Conflicts []string
}

type entry struct {
	rule   Rule
	when   Cond
	origin string
}

type wild struct {
	key Sym
	acc *Accept
}

type compiler struct {
	t       *Table
	g       *Grammar
	rules   []Rule
	anys    []Any
	entries []entry
	sig     []string
	index   map[string]int
}

func Compile(g *Grammar, a *Alphabet) (*Table, error) {
	if err := Check(g); err != nil {
		return nil, err
	}
	g, err := expand(g)
	if err != nil {
		return nil, err
	}
	rules := make([]Rule, len(g.Rules))
	for i, r := range g.Rules {
		r.Elems = append([]Class(nil), r.Elems...)
		for j, c := range r.Elems {
			got, err := resolve(c, a)
			if err != nil {
				return nil, fmt.Errorf("%s:%d: %w", g.Name, r.Line, err)
			}
			r.Elems[j] = got
		}
		rules[i] = r
	}
	anys := make([]Any, len(g.Any))
	for i, x := range g.Any {
		got, err := resolve(x.Elem, a)
		if err != nil {
			return nil, fmt.Errorf("%s:%d: %w", g.Name, x.Line, err)
		}
		key, ok := got.only()
		if !ok {
			return nil, fmt.Errorf("%s:%d: `... %s` is not one symbol: a wildcard cannot say which arrived", g.Name, x.Line, x.Pattern)
		}
		x.Elem, x.key = got, key
		anys[i] = x
	}
	for scope, name := range g.Settles {
		if a.Of(name) == 0 {
			return nil, fmt.Errorf("%s: %s %%settles %s, and nothing declares %s", g.Name, scope, name, name)
		}
	}

	t := &Table{Name: g.Name, Note: g.Note, Alpha: a, Delays: g.Delays}
	c := &compiler{t: t, g: g, rules: rules, anys: anys, index: map[string]int{}}
	for i, name := range positions(g) {
		t.Scopes = append(t.Scopes, Scope{Name: name})
		c.index[name] = i + 1
	}
	c.state("")
	for i := range t.Scopes {
		if err := c.scope(i); err != nil {
			return nil, err
		}
	}
	sort.Strings(t.Conflicts)
	t.Conflicts = slices.Compact(t.Conflicts)
	return c.minimal(), nil
}

func positions(g *Grammar) []string {
	at := map[string]bool{g.Scopes[0]: true}
	for _, r := range g.Rules {
		if r.Goto != "" {
			at[r.Goto] = true
		}
	}
	for _, x := range g.Any {
		if x.Goto != "" {
			at[x.Goto] = true
		}
	}
	var out []string
	for _, s := range g.Scopes {
		if at[s] {
			out = append(out, s)
		}
	}
	return out
}

func (c *compiler) minimal() *Table {
	t := c.t
	var starts []int
	for _, scope := range t.Scopes {
		starts = append(starts, scope.Start)
	}
	to := fa.Reachable(t.Next, fa.Equivalent(t.Next, c.sig), starts...)
	n := 0
	for _, s := range to {
		n = max(n, s)
	}
	out := &Table{Name: t.Name, Note: t.Note, Alpha: t.Alpha, Delays: t.Delays, Conflicts: t.Conflicts,
		Next: make([][]int, n+1), Accept: make([][]*Accept, n+1), Live: make([][]Cond, n+1),
		Settle: make([]Sym, n+1), Any: make([][][]*Accept, n+1)}
	for s := range t.Next {
		if s != 0 && to[s] == 0 {
			continue
		}
		m := to[s]
		out.Next[m] = make([]int, len(t.Next[s]))
		for sym, n := range t.Next[s] {
			out.Next[m][sym] = to[n]
		}
		out.Accept[m], out.Live[m], out.Settle[m], out.Any[m] = t.Accept[s], t.Live[s], t.Settle[s], t.Any[s]
	}
	for _, scope := range t.Scopes {
		out.Scopes = append(out.Scopes, Scope{Name: scope.Name, Start: to[scope.Start]})
	}
	return out
}

func (c *compiler) scope(i int) error {
	scope := c.t.Scopes[i].Name
	first := len(c.entries)
	var wilds []wild
	wildcard := func(x Any, when Cond) {
		wilds = append(wilds, wild{x.key, &Accept{Action: x.Action, Goto: c.index[x.Goto], When: when, Pattern: "... " + x.Pattern, Line: x.Line}})
	}
	var lower func(at string, when Cond, origin string)
	lower = func(at string, when Cond, origin string) {
		for i, r := range c.rules {
			switch {
			case r.Scope != at:
			case r.Include != "":
				lower(r.Include, AllOf(when, r.When), fmt.Sprintf("%s -%d-> %s", origin, i, r.Include))
			default:
				c.entries = append(c.entries, entry{rule: r, when: AllOf(when, r.When), origin: origin})
			}
		}
		for _, x := range c.anys {
			if x.Scope == at {
				wildcard(x, AllOf(when, x.When))
			}
		}
	}
	lower(scope, nil, scope)
	for _, x := range c.anys {
		if x.Scope == "" {
			wildcard(x, x.When)
		}
	}

	start, err := c.build(scope, c.entries[first:], first, wilds)
	c.t.Scopes[i].Start = start
	return err
}

func (c *compiler) build(scope string, entries []entry, first int, wilds []wild) (int, error) {
	t := c.t
	rules := make([]fa.Rule[Sym], len(entries))
	for i, e := range entries {
		var es []fa.Expr[Sym]
		for _, el := range e.rule.Elems {
			x := fa.Lit(el.set())
			if el.Many {
				x = fa.Plus(x)
			}
			es = append(es, x)
		}
		rules[i] = fa.Rule[Sym]{Expr: fa.Cat(es...), Name: e.rule.Pattern}
	}
	every := fa.Span(Sym(1), Sym(t.Alpha.Size()))
	d, err := fa.BuildWith(rules, fa.Partition(append(fa.Sets(rules), every)))
	if err != nil {
		return 0, fmt.Errorf("%s: %s: %w", c.g.Name, scope, err)
	}

	base := len(t.Next) - 1
	at := func(s fa.State) int {
		if s == fa.NoState {
			return 0
		}
		return base + int(s)
	}
	for s := fa.State(1); int(s) <= d.States(); s++ {
		n := c.state("")
		for sym := Sym(1); int(sym) <= t.Alpha.Size(); sym++ {
			t.Next[n][sym] = at(d.Step(s, sym))
		}
		for _, w := range wilds {
			t.Any[n][w.key] = append(t.Any[n][w.key], w.acc)
		}
		done, going := shift(d.Done(s), first), shift(d.Going(s), first)
		c.finish(n, done, going)
		c.sig[n] = signature(done, going, wilds)
	}
	return at(d.Start()), nil
}

func shift(ms []fa.Match, first int) []int {
	out := make([]int, len(ms))
	for i, m := range ms {
		out[i] = first + int(m) - 1
	}
	return out
}

func (c *compiler) finish(s int, done, going []int) {
	t := c.t
	live := map[string][]string{}
	for _, e := range going {
		t.Live[s] = append(t.Live[s], c.entries[e].when)
		live[c.entries[e].origin] = append(live[c.entries[e].origin], c.entries[e].rule.Pattern)
	}
	finished := map[string]*Accept{}
	for _, i := range done {
		e := c.entries[i]
		acc := &Accept{Action: e.rule.Action, Goto: c.index[e.rule.Goto], When: e.when,
			Pattern: e.rule.Pattern, Line: e.rule.Line, Operand: operand(e.rule)}
		if was, ok := finished[e.origin]; ok {
			c.conflict(acc.Line, "%q and %q both finish here -- two rules, one key sequence", was.Pattern, acc.Pattern)
			continue
		}
		finished[e.origin] = acc
		t.Accept[s] = append(t.Accept[s], acc)
		if len(going) == 0 {
			continue
		}
		name, ok := c.g.Settles[e.rule.Scope]
		if !ok && len(live[e.origin]) == 0 {
			continue
		}
		if !ok {
			c.conflict(acc.Line, "%q is complete, and %s can still follow -- %s has no %%settles to say which was meant",
				acc.Pattern, strings.Join(live[e.origin], ", "), e.rule.Scope)
			continue
		}
		switch sym := t.Alpha.Of(name); t.Settle[s] {
		case 0:
			t.Settle[s] = sym
		case sym:
		default:
			c.conflict(acc.Line, "%q settles on %s, and another rule here on %s", acc.Pattern, name, t.Alpha.Name(t.Settle[s]))
		}
	}
	if settle := t.Settle[s]; settle != 0 && t.Next[s][settle] != 0 {
		c.conflict(t.Accept[s][0].Line, "%s settles here and a rule reads %s next -- which is meant", t.Alpha.Name(settle), t.Alpha.Name(settle))
	}
}

func (c *compiler) conflict(line int, f string, args ...any) {
	c.t.Conflicts = append(c.t.Conflicts, fmt.Sprintf("%s:%d: %s", c.g.Name, line, fmt.Sprintf(f, args...)))
}

func resolve(c Class, a *Alphabet) (Class, error) {
	if c.Event == "" {
		return c, nil
	}
	s := a.Of(c.Event)
	if s == 0 {
		return c, fmt.Errorf("%q is in a pattern and nothing declares it", c.Event)
	}
	c.sym = s
	return c, nil
}

func operand(r Rule) *Slice {
	keys := 0
	for _, c := range r.Elems {
		if !c.IsEvent() {
			keys++
		}
	}
	var o *Slice
	at := 0
	for _, c := range r.Elems {
		if c.IsEvent() {
			continue
		}
		if c.wide() {
			if o == nil {
				o = &Slice{From: at}
			}
			o.To = at - keys + 1
		}
		at++
	}
	return o
}

func signature(done, going []int, wilds []wild) string {
	var b strings.Builder
	fmt.Fprint(&b, done, going)
	for _, w := range wilds {
		fmt.Fprintf(&b, " %p", w.acc)
	}
	return b.String()
}

func (c *compiler) state(sig string) int {
	c.sig = append(c.sig, sig)
	return c.t.state()
}

func (t *Table) state() int {
	s := len(t.Next)
	t.Next = append(t.Next, make([]int, t.Alpha.Size()+1))
	t.Any = append(t.Any, make([][]*Accept, t.Alpha.Size()+1))
	t.Accept = append(t.Accept, nil)
	t.Live = append(t.Live, nil)
	t.Settle = append(t.Settle, 0)
	return s
}

func (t *Table) Scope(name string) int {
	for i, s := range t.Scopes {
		if s.Name == name {
			return i + 1
		}
	}
	return 0
}

func (t *Table) Waiting() []string {
	var out []string
	for name := range t.Delays {
		if sym := t.Alpha.Of(name); sym != 0 && t.waits(sym) {
			out = append(out, name)
		}
	}
	sort.Strings(out)
	return out
}

func (t *Table) waits(sym Sym) bool {
	for s := range t.Next {
		if t.Next[s][sym] != 0 || len(t.Any[s][sym]) > 0 || t.Settle[s] == sym {
			return true
		}
	}
	return false
}
