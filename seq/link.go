package seq

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

func Link(tabs []*Table, name, handoff string) (*Table, error) {
	out := &Table{Name: name, Start: map[string]int{}, Roots: map[string]string{}}
	seen := map[string]bool{}
	for _, t := range tabs {
		if seen[t.Name] {
			return nil, fmt.Errorf("two trees are called %s", t.Name)
		}
		seen[t.Name] = true
	}
	tabs = rooted(tabs)
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
	if out.Handoff = out.Alpha.Of(handoff); out.Handoff == 0 {
		return nil, fmt.Errorf("the alphabet has no %s to hand off on", handoff)
	}

	out.state("")
	var missing []string
	for _, t := range tabs {
		base := len(out.Next) - 1
		at := func(scope string) string { return t.Name + "." + scope }
		moved := func(as []*Accept) []*Accept {
			out := make([]*Accept, len(as))
			for i, a := range as {
				b := *a
				if b.Goto != "" {
					b.Goto = at(b.Goto)
				}
				if b.Hands != "" && !seen[b.Hands] {
					missing = append(missing, fmt.Sprintf("%s:%d: %q hands to %q, and there is no such tree", t.Name, a.Line, a.Pattern, a.Hands))
				}
				out[i] = &b
			}
			return out
		}
		for s := 1; s < len(t.Next); s++ {
			n := out.state(at(t.Scope[s]))
			for sym, to := range t.Next[s] {
				if to != 0 {
					out.Next[n][sym] = to + base
				}
				out.Any[n][sym] = moved(t.Any[s][sym])
			}
			out.Accept[n] = moved(t.Accept[s])
			out.Live[n] = t.Live[s]
			out.Settle[n] = t.Settle[s]
		}
		for _, scope := range t.Scopes {
			out.Scopes = append(out.Scopes, at(scope))
			out.Start[at(scope)] = t.Start[scope] + base
		}
		if len(t.Scopes) > 0 {
			out.Roots[t.Name] = at(t.Scopes[0])
		}
		for name, d := range t.Delays {
			if out.Delays == nil {
				out.Delays = map[string]time.Duration{}
			}
			if was, ok := out.Delays[name]; ok && was != d {
				return nil, fmt.Errorf("%s waits %s for %s, and another tree %s", t.Name, d, name, was)
			}
			out.Delays[name] = d
		}
		out.Conflicts = append(out.Conflicts, t.Conflicts...)
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return nil, fmt.Errorf("%s", strings.Join(missing, "\n"))
	}
	return out, nil
}

func rooted(tabs []*Table) []*Table {
	by := map[string]*Table{}
	handed := map[string]bool{}
	for _, t := range tabs {
		by[t.Name] = t
		for _, a := range hands(t) {
			handed[a] = true
		}
	}
	var out []*Table
	done := map[*Table]bool{}
	var walk func(t *Table)
	walk = func(t *Table) {
		if t == nil || done[t] {
			return
		}
		done[t] = true
		out = append(out, t)
		for _, h := range hands(t) {
			walk(by[h])
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

func hands(t *Table) []string {
	var out []string
	for _, as := range t.Accept {
		for _, a := range as {
			if a.Hands != "" {
				out = append(out, a.Hands)
			}
		}
	}
	return out
}
