package seq

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

type Slice struct {
	From int
	To   int
}

type Accept struct {
	Action  string
	Goto    string
	Hands   string
	When    Cond
	Rule    int
	Operand *Slice
}

type Table struct {
	Name      string
	Note      string
	Alpha     *Alphabet
	Rules     []Rule
	Start     map[string]int
	Scope     []string
	Next      [][]int
	Accept    []*Accept
	Extend    []bool
	Any       [][]*Accept
	Delays    map[string]time.Duration
	Conflicts []string
}

type item struct{ r, i int }

func Compile(g *Grammar, a *Alphabet) (*Table, error) {
	rules := make([]Rule, len(g.Rules))
	for i, r := range g.Rules {
		r.Elems = append([]Class(nil), r.Elems...)
		for j, c := range r.Elems {
			got, err := resolve(c, a)
			if err != nil {
				return nil, fmt.Errorf("%s:%d: %s", g.Name, r.Line, err)
			}
			r.Elems[j] = got
		}
		rules[i] = r
	}

	anys := make([]Any, len(g.Any))
	for i, x := range g.Any {
		got, err := resolve(x.Elem, a)
		if err != nil {
			return nil, fmt.Errorf("%s:%d: %s", g.Name, x.Line, err)
		}
		key, ok := got.only()
		if !ok {
			return nil, fmt.Errorf("%s:%d: `... %s` is not one symbol: a wildcard cannot say which arrived", g.Name, x.Line, x.Pattern)
		}
		x.Elem, x.key = got, key
		anys[i] = x
	}

	t := &Table{Name: g.Name, Note: g.Note, Alpha: a, Rules: rules, Start: map[string]int{}, Delays: g.Delays}
	byScope := map[string][]int{}
	for i, r := range rules {
		byScope[r.Scope] = append(byScope[r.Scope], i)
	}

	seen := map[string]int{}
	var add func(scope string, items []item) int
	add = func(scope string, items []item) int {
		sort.Slice(items, func(x, y int) bool {
			if items[x].r != items[y].r {
				return items[x].r < items[y].r
			}
			return items[x].i < items[y].i
		})
		key := fmt.Sprint(items)
		if s, ok := seen[key]; ok {
			return s
		}
		s := t.state(scope, nil)
		seen[key] = s

		for _, it := range items {
			r := rules[it.r]
			if it.i < len(r.Elems) {
				continue
			}
			acc := &Accept{Action: r.Action, Goto: r.Goto, Hands: r.Hands, When: r.When, Rule: it.r, Operand: operand(r)}
			if t.Accept[s] == nil {
				t.Accept[s] = acc
				continue
			}
			t.Conflicts = append(t.Conflicts, fmt.Sprintf("state %d: %q and %q both finish here -- two rules, one key sequence",
				s, rules[t.Accept[s].Rule].Pattern, r.Pattern))
		}

		for sym := Sym(0); sym < a.Size(); sym++ {
			var next []item
			for _, it := range items {
				r := rules[it.r]
				if it.i >= len(r.Elems) || !r.Elems[it.i].has(sym) {
					continue
				}
				if r.Elems[it.i].Many {
					next = append(next, it)
				}
				next = append(next, item{it.r, it.i + 1})
			}
			if len(next) == 0 {
				continue
			}
			if a.IsKey(sym) {
				t.Extend[s] = true
			}
			t.Next[s][sym] = add(scope, next)
		}
		return s
	}

	for _, scope := range g.Scopes {
		var items []item
		for _, r := range byScope[scope] {
			items = append(items, item{r, 0})
		}
		t.Start[scope] = add(scope, items)
	}

	if err := t.settle(g); err != nil {
		return nil, err
	}
	t.ambiguous()
	t.anys(anys)
	return t, nil
}

func resolve(c Class, a *Alphabet) (Class, error) {
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

func (t *Table) state(scope string, a *Accept) int {
	s := len(t.Next)
	row := make([]int, t.Alpha.Size())
	for i := range row {
		row[i] = Dead
	}
	t.Next = append(t.Next, row)
	t.Scope = append(t.Scope, scope)
	t.Accept = append(t.Accept, a)
	t.Extend = append(t.Extend, false)
	return s
}

func (t *Table) settle(g *Grammar) error {
	for s := range t.Next {
		a := t.Accept[s]
		if a == nil || !t.Extend[s] {
			continue
		}
		name, ok := g.Settles[t.Scope[s]]
		if !ok {
			continue
		}
		sym, ok := t.Alpha.Of(name)
		if !ok {
			return fmt.Errorf("%s: %%settles %s, and nothing declares %s", t.Scope[s], name, name)
		}
		if t.Next[s][sym] != Dead {
			continue
		}
		t.Next[s][sym] = t.state(t.Scope[s], a)
		t.Accept[s] = nil
	}
	return nil
}

func (t *Table) ambiguous() {
	for s, a := range t.Accept {
		if a == nil || !t.Extend[s] {
			continue
		}
		var on []string
		for sym := Sym(0); sym < t.Alpha.Keys(); sym++ {
			if t.Next[s][sym] != Dead {
				on = append(on, t.Alpha.Name(sym))
			}
		}
		t.Conflicts = append(t.Conflicts, fmt.Sprintf("state %d: %q is complete, and %s can still follow -- %s has no %%settles to say which was meant",
			s, t.Rules[a.Rule].Pattern, strings.Join(on, ", "), t.Scope[s]))
	}
}

func (t *Table) anys(anys []Any) {
	t.Any = make([][]*Accept, len(t.Next))
	for s := range t.Any {
		t.Any[s] = make([]*Accept, t.Alpha.Size())
		for _, x := range anys {
			if x.Scope != "" && x.Scope != t.Scope[s] {
				continue
			}
			if t.Next[s][x.key] != Dead {
				continue
			}
			if x.Scope == "" && t.Any[s][x.key] != nil {
				continue
			}
			t.Any[s][x.key] = &Accept{Action: x.Action, Goto: x.Goto, When: x.When, Rule: -1}
		}
	}
}

func (t *Table) Waiting() []string {
	var out []string
	for name := range t.Delays {
		if t.WaitsIn(name) >= 0 {
			out = append(out, name)
		}
	}
	sort.Strings(out)
	return out
}

func (t *Table) WaitsIn(name string) int {
	sym, ok := t.Alpha.Of(name)
	if !ok {
		return -1
	}
	for s, row := range t.Next {
		if row[sym] != Dead {
			return s
		}
	}
	return -1
}
