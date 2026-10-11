package keypad

import (
	"errors"
	"fmt"
	"time"

	"github.com/cptaffe/automaton/seq"
)

func Link(trees ...*Tree) (*seq.Grammar, error) {
	by := map[string]*Tree{}
	handed := map[string]bool{}
	for _, t := range trees {
		if by[t.Grammar.Name] != nil {
			return nil, fmt.Errorf("two trees are called %s", t.Grammar.Name)
		}
		by[t.Grammar.Name] = t
		for _, to := range t.hands {
			handed[to] = true
		}
	}
	order := make([]*Tree, 0, len(trees))
	for _, t := range trees {
		if !handed[t.Grammar.Name] {
			order = append(order, t)
		}
	}
	for _, t := range trees {
		if handed[t.Grammar.Name] {
			order = append(order, t)
		}
	}

	out := &seq.Grammar{Name: "trees", Delays: map[string]time.Duration{}, Settles: map[string]string{}}
	if len(trees) == 1 {
		out.Name, out.Note = trees[0].Grammar.Name, trees[0].Grammar.Note
	}
	var errs []error
	waits := map[string]bool{}
	for _, t := range order {
		g := t.Grammar
		at := func(scope string) string { return g.Name + "." + scope }
		for _, s := range g.Scopes {
			out.Scopes = append(out.Scopes, at(s))
		}
		for name, d := range g.Delays {
			if was, ok := out.Delays[name]; ok && was != d {
				errs = append(errs, fmt.Errorf("%s waits %s for %s, and another tree %s", g.Name, d, name, was))
			}
			out.Delays[name] = d
		}
		for s, name := range g.Settles {
			out.Settles[at(s)] = name
		}
		for i, r := range g.Rules {
			r.Scope = at(r.Scope)
			if r.Goto != "" {
				r.Goto = at(r.Goto)
			}
			if r.Include != "" {
				r.Include = at(r.Include)
			}
			if to, ok := t.hands[i]; ok {
				next := by[to]
				if next == nil {
					errs = append(errs, fmt.Errorf("%s:%d: %q hands to %s, and there is no such tree", g.Name, r.Line, r.Pattern, to))
					continue
				}
				r.Goto = "=> " + to
				if !waits[r.Goto] {
					waits[r.Goto] = true
					out.Scopes = append(out.Scopes, r.Goto)
					out.Rules = append(out.Rules,
						seq.Rule{Scope: r.Goto, Pattern: Answered, Elems: []seq.Class{{Event: Answered, Text: Answered}},
							Goto: to + "." + next.Grammar.Scopes[0], Line: r.Line})
				}
			}
			out.Rules = append(out.Rules, r)
		}
		for _, x := range g.Any {
			if x.Goto != "" {
				x.Goto = at(x.Goto)
			}
			scopes := []string{x.Scope}
			if x.Scope == "" {
				scopes = g.Scopes
			}
			for _, s := range scopes {
				x.Scope = at(s)
				out.Any = append(out.Any, x)
			}
		}
	}
	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}
	return out, seq.Check(out)
}

func Compile(trees ...*Tree) (*seq.Table, error) {
	g, err := Link(trees...)
	if err != nil {
		return nil, err
	}
	return seq.Compile(g, Alphabet(g))
}
