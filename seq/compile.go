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
	Ahead   int
	Operand *Slice
}

type Scope struct {
	Name  string
	Start int
}

type State struct {
	Next   []int
	Any    [][]*Accept
	Accept []*Accept
	Live   []Cond
}

type Table struct {
	Name      string
	Note      string
	Alpha     *Alphabet
	Classes   *fa.Classes[Sym]
	Scopes    []Scope
	States    []State
	Delays    map[string]time.Duration
	Conflicts []string
}

func (t *Table) next() [][]int {
	out := make([][]int, len(t.States))
	for i := range t.States {
		out[i] = t.States[i].Next
	}
	return out
}

type entry struct {
	rule   Rule
	when   Cond
	origin string
}

type wild struct {
	keys fa.Set[Sym]
	acc  *Accept
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
	c := &compiler{g: g, index: map[string]int{}}
	if err := c.bind(a); err != nil {
		return nil, err
	}
	t := &Table{Name: g.Name, Note: g.Note, Alpha: a, Classes: c.partition(a), Delays: g.Delays}
	c.t = t
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

func (c *compiler) bind(a *Alphabet) error {
	g := c.g
	c.rules = make([]Rule, len(g.Rules))
	for i, r := range g.Rules {
		r.Elems = append([]Class(nil), r.Elems...)
		for j, el := range r.Elems {
			got, err := resolve(el, a)
			if err != nil {
				return fmt.Errorf("%s:%d: %w", g.Name, r.Line, err)
			}
			r.Elems[j] = got
		}
		c.rules[i] = r
	}
	c.anys = make([]Any, len(g.Any))
	for i, x := range g.Any {
		got, err := resolve(x.Elem, a)
		if err != nil {
			return fmt.Errorf("%s:%d: %w", g.Name, x.Line, err)
		}
		if got.Many {
			return fmt.Errorf("%s:%d: `... %s` repeats, and a wildcard reads one key", g.Name, x.Line, x.Pattern)
		}
		x.Elem = got
		c.anys[i] = x
	}
	return nil
}

func (c *compiler) partition(a *Alphabet) *fa.Classes[Sym] {
	sets := []fa.Set[Sym]{fa.Span(0, Sym(a.Size()-1))}
	for _, r := range c.rules {
		for _, el := range r.Elems {
			sets = append(sets, el.set())
		}
	}
	for _, x := range c.anys {
		sets = append(sets, x.Elem.set())
	}
	return fa.Partition(sets)
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
	starts := make([]int, 0, len(t.Scopes))
	for _, scope := range t.Scopes {
		starts = append(starts, scope.Start)
	}
	next := t.next()
	to := fa.Reachable(next, fa.Equivalent(next, c.sig), starts...)
	n := 0
	for _, s := range to {
		n = max(n, s)
	}
	out := &Table{Name: t.Name, Note: t.Note, Alpha: t.Alpha, Classes: t.Classes, Delays: t.Delays, Conflicts: t.Conflicts, States: make([]State, n+1)}
	for s, st := range t.States {
		if s != 0 && to[s] == 0 {
			continue
		}
		st.Next = make([]int, len(next[s]))
		for sym, n := range next[s] {
			st.Next[sym] = to[n]
		}
		out.States[to[s]] = st
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
		wilds = append(wilds, wild{x.Elem.set(), &Accept{Action: x.Action, Goto: c.index[x.Goto], When: when, Pattern: "... " + x.Pattern, Line: x.Line}})
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
			if el.Any {
				x = fa.Fallback(el.set())
			}
			if el.Many {
				x = fa.Plus(x)
			}
			es = append(es, x)
		}
		rules[i] = fa.Rule[Sym]{Expr: fa.Cat(es...), Name: e.rule.Pattern}
	}
	d, err := fa.BuildWith(rules, t.Classes)
	if err != nil {
		return 0, fmt.Errorf("%s: %s: %w", c.g.Name, scope, err)
	}

	base := len(t.States) - 1
	at := func(s fa.State) int {
		if s == fa.NoState {
			return 0
		}
		return base + int(s)
	}
	for s := fa.State(1); int(s) <= d.States(); s++ {
		n := c.state("")
		for k := fa.Class(1); int(k) <= t.Classes.Len(); k++ {
			t.States[n].Next[k] = at(d.Next(s, k))
		}
		for _, w := range wilds {
			for _, k := range t.Classes.Covers(w.keys) {
				t.States[n].Any[k] = append(t.States[n].Any[k], w.acc)
			}
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
	st := &t.States[s]
	live := map[string][]string{}
	for _, e := range going {
		st.Live = append(st.Live, c.entries[e].when)
		live[c.entries[e].origin] = append(live[c.entries[e].origin], c.entries[e].rule.Pattern)
	}
	finished := map[string]*Accept{}
	for _, i := range done {
		e := c.entries[i]
		acc := &Accept{Action: e.rule.Action, Goto: c.index[e.rule.Goto], When: e.when,
			Pattern: e.rule.Pattern, Line: e.rule.Line, Ahead: e.rule.Ahead, Operand: operand(e.rule)}
		if was, ok := finished[e.origin]; ok {
			c.conflict(acc.Line, "%q and %q both finish here -- two rules, one key sequence", was.Pattern, acc.Pattern)
			continue
		}
		finished[e.origin] = acc
		st.Accept = append(st.Accept, acc)
		if len(live[e.origin]) > 0 {
			c.conflict(acc.Line, "%q is complete, and %s can still follow -- end it with a delay, or peek at the key after it",
				acc.Pattern, strings.Join(live[e.origin], ", "))
		}
	}
}

func (c *compiler) conflict(line int, f string, args ...any) {
	c.t.Conflicts = append(c.t.Conflicts, fmt.Sprintf("%s:%d: %s", c.g.Name, line, fmt.Sprintf(f, args...)))
}

func resolve(c Class, a *Alphabet) (Class, error) {
	if c.Any {
		c.keys = fa.Span(0, Sym(a.Keys()-1))
		return c, nil
	}
	if c.Event == "" {
		return c, nil
	}
	s, ok := a.Of(c.Event)
	if !ok {
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
	s := len(t.States)
	n := t.Classes.Len() + 1
	t.States = append(t.States, State{Next: make([]int, n), Any: make([][]*Accept, n)})
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
		if sym, ok := t.Alpha.Of(name); ok && t.waits(sym) {
			out = append(out, name)
		}
	}
	sort.Strings(out)
	return out
}

func (t *Table) waits(sym Sym) bool {
	k := t.Classes.Of(sym)
	for _, st := range t.States {
		if st.Next[k] != 0 || len(st.Any[k]) > 0 {
			return true
		}
	}
	return false
}
