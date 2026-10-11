package seq

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/cptaffe/automaton"
)

type Slice struct {
	From int
	To   int
}

type Accept struct {
	Action  string
	Goto    string
	When    Cond
	Pattern string
	Line    int
	Operand *Slice
}

type Table struct {
	Name      string
	Note      string
	Alpha     *Alphabet
	Scopes    []string
	Start     map[string]int
	Scope     []string
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

type item struct{ e, i int }

type compiler struct {
	t       *Table
	g       *Grammar
	rules   []Rule
	anys    []Any
	entries []entry
	seen    map[string]int
	sig     []string
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

	t := &Table{Name: g.Name, Note: g.Note, Alpha: a, Scopes: positions(g), Start: map[string]int{}, Delays: g.Delays}
	c := &compiler{t: t, g: g, rules: rules, anys: anys, seen: map[string]int{}}
	c.state("", "")
	for _, scope := range t.Scopes {
		c.scope(scope)
	}
	sort.Strings(t.Conflicts)
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
		starts = append(starts, t.Start[scope])
	}
	to := automaton.Reachable(t.Next, automaton.Equivalent(t.Next, c.sig), starts...)
	n := 0
	for _, s := range to {
		n = max(n, s)
	}
	out := &Table{Name: t.Name, Note: t.Note, Alpha: t.Alpha, Scopes: t.Scopes, Start: map[string]int{}, Delays: t.Delays, Conflicts: t.Conflicts,
		Scope: make([]string, n+1), Next: make([][]int, n+1), Accept: make([][]*Accept, n+1), Live: make([][]Cond, n+1),
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
		out.Scope[m], out.Accept[m], out.Live[m], out.Settle[m], out.Any[m] = t.Scope[s], t.Accept[s], t.Live[s], t.Settle[s], t.Any[s]
	}
	for scope, s := range t.Start {
		out.Start[scope] = to[s]
	}
	return out
}

func (c *compiler) scope(scope string) {
	first := len(c.entries)
	var wilds []wild
	wildcard := func(x Any, when Cond) {
		wilds = append(wilds, wild{x.key, &Accept{Action: x.Action, Goto: x.Goto, When: when, Pattern: "... " + x.Pattern, Line: x.Line}})
	}
	var lower func(at string, when Cond, origin string)
	lower = func(at string, when Cond, origin string) {
		for _, r := range c.rules {
			switch {
			case r.Scope != at:
			case r.Include != "":
				lower(r.Include, AllOf(when, r.When), origin+" -> "+r.Include)
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

	var items []item
	for e := first; e < len(c.entries); e++ {
		items = append(items, item{e, 0})
	}
	c.t.Start[scope] = c.add(scope, items, wilds)
}

func (c *compiler) add(scope string, items []item, wilds []wild) int {
	sort.Slice(items, func(x, y int) bool {
		if items[x].e != items[y].e {
			return items[x].e < items[y].e
		}
		return items[x].i < items[y].i
	})
	key := scope + fmt.Sprint(items)
	if s, ok := c.seen[key]; ok {
		return s
	}
	t := c.t
	s := c.state(scope, signature(items, wilds, c.entries))
	c.seen[key] = s

	c.finish(s, items)
	for _, w := range wilds {
		t.Any[s][w.key] = append(t.Any[s][w.key], w.acc)
	}
	for sym := Sym(1); int(sym) <= t.Alpha.Size(); sym++ {
		var next []item
		for _, it := range items {
			r := c.entries[it.e].rule
			if it.i >= len(r.Elems) || !r.Elems[it.i].has(sym) {
				continue
			}
			if r.Elems[it.i].Many {
				next = append(next, it)
			}
			next = append(next, item{it.e, it.i + 1})
		}
		if len(next) > 0 {
			t.Next[s][sym] = c.add(scope, next, wilds)
		}
	}
	return s
}

func (c *compiler) finish(s int, items []item) {
	t := c.t
	going := map[string][]string{}
	for _, it := range items {
		if e := c.entries[it.e]; it.i < len(e.rule.Elems) {
			t.Live[s] = append(t.Live[s], e.when)
			going[e.origin] = append(going[e.origin], e.rule.Pattern)
		}
	}
	done := map[string]*Accept{}
	for _, it := range items {
		e := c.entries[it.e]
		if it.i < len(e.rule.Elems) {
			continue
		}
		acc := &Accept{Action: e.rule.Action, Goto: e.rule.Goto, When: e.when,
			Pattern: e.rule.Pattern, Line: e.rule.Line, Operand: operand(e.rule)}
		if was, ok := done[e.origin]; ok {
			c.conflict(acc.Line, "%q and %q both finish here -- two rules, one key sequence", was.Pattern, acc.Pattern)
			continue
		}
		done[e.origin] = acc
		t.Accept[s] = append(t.Accept[s], acc)
		if len(going[e.origin]) == 0 {
			continue
		}
		name, ok := c.g.Settles[e.rule.Scope]
		if !ok {
			c.conflict(acc.Line, "%q is complete, and %s can still follow -- %s has no %%settles to say which was meant",
				acc.Pattern, strings.Join(going[e.origin], ", "), e.rule.Scope)
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

func signature(items []item, wilds []wild, entries []entry) string {
	var b strings.Builder
	seen := map[string]bool{}
	for _, it := range items {
		k := fmt.Sprintf("%d:%t ", it.e, it.i >= len(entries[it.e].rule.Elems))
		if !seen[k] {
			seen[k] = true
			b.WriteString(k)
		}
	}
	for _, w := range wilds {
		fmt.Fprintf(&b, "%p ", w.acc)
	}
	return b.String()
}

func (c *compiler) state(scope, sig string) int {
	c.sig = append(c.sig, sig)
	return c.t.state(scope)
}

func (t *Table) state(scope string) int {
	s := len(t.Next)
	t.Next = append(t.Next, make([]int, t.Alpha.Size()+1))
	t.Any = append(t.Any, make([][]*Accept, t.Alpha.Size()+1))
	t.Scope = append(t.Scope, scope)
	t.Accept = append(t.Accept, nil)
	t.Live = append(t.Live, nil)
	t.Settle = append(t.Settle, 0)
	return s
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
