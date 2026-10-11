package seq

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

func Link(tabs []*Table, name, handoff string) (*Table, error) {
	tabs = rooted(tabs)
	out := &Table{Name: name, Start: map[string]int{}}
	for _, t := range tabs {
		if out.Alpha == nil {
			out.Alpha = t.Alpha
		}
		if t.Alpha != out.Alpha {
			return nil, fmt.Errorf("%s reads a different alphabet: one link, one alphabet", t.Name)
		}
	}
	if out.Alpha == nil {
		return out, nil
	}
	on, ok := out.Alpha.Of(handoff)
	if !ok {
		return nil, fmt.Errorf("the alphabet has no %s to hand off on", handoff)
	}

	at := map[string]int{}
	first := map[string]int{}
	for _, t := range tabs {
		at[t.Name] = len(out.Next)
		base := len(out.Rules)
		low := -1
		for _, s := range t.Start {
			if low < 0 || s < low {
				low = s
			}
		}
		first[t.Name] = at[t.Name] + low
		for s := range t.Next {
			row := make([]int, out.Alpha.Size())
			any := make([]*Accept, out.Alpha.Size())
			for sym := range row {
				row[sym] = Dead
				if n := t.Next[s][sym]; n != Dead {
					row[sym] = n + at[t.Name]
				}
				any[sym] = moved(t.Any[s][sym], t.Name, base)
			}
			out.Next = append(out.Next, row)
			out.Any = append(out.Any, any)
			out.Extend = append(out.Extend, t.Extend[s])
			out.Scope = append(out.Scope, t.Name+"."+t.Scope[s])
			out.Accept = append(out.Accept, moved(t.Accept[s], t.Name, base))
		}
		for _, r := range t.Rules {
			r.Scope = t.Name + "." + r.Scope
			if r.Goto != "" {
				r.Goto = t.Name + "." + r.Goto
			}
			out.Rules = append(out.Rules, r)
		}
		for s, n := range t.Start {
			out.Start[t.Name+"."+s] = n + at[t.Name]
		}
		for name, d := range t.Delays {
			if out.Delays == nil {
				out.Delays = map[string]time.Duration{}
			}
			out.Delays[name] = d
		}
		out.Conflicts = append(out.Conflicts, t.Conflicts...)
	}

	var missing []string
	for s, a := range out.Accept {
		if a == nil || a.Hands == "" {
			continue
		}
		to, ok := first[a.Hands]
		if !ok {
			missing = append(missing, fmt.Sprintf("%q hands to %q, and there is no such tree", out.Rules[a.Rule].Pattern, a.Hands))
			continue
		}
		out.Next[s][on] = to
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return nil, fmt.Errorf("%s", strings.Join(missing, "; "))
	}
	return out, nil
}

func rooted(tabs []*Table) []*Table {
	by := map[string]*Table{}
	handed := map[string]bool{}
	for _, t := range tabs {
		by[t.Name] = t
		for _, a := range t.Accept {
			if a != nil && a.Hands != "" {
				handed[a.Hands] = true
			}
		}
	}
	var out []*Table
	done := map[string]bool{}
	var walk func(t *Table)
	walk = func(t *Table) {
		if t == nil || done[t.Name] {
			return
		}
		done[t.Name] = true
		out = append(out, t)
		for _, a := range t.Accept {
			if a != nil && a.Hands != "" {
				walk(by[a.Hands])
			}
		}
	}
	for _, t := range tabs {
		if !handed[t.Name] {
			walk(t)
		}
	}
	for _, t := range tabs {
		walk(t)
	}
	return out
}

func moved(a *Accept, tree string, base int) *Accept {
	if a == nil {
		return nil
	}
	b := *a
	if b.Goto != "" {
		b.Goto = tree + "." + b.Goto
	}
	if b.Rule >= 0 {
		b.Rule += base
	}
	return &b
}
