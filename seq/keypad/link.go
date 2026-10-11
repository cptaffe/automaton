package keypad

import (
	"fmt"
	"maps"
	"slices"
	"time"

	"github.com/cptaffe/automaton/seq"
)

func Link(trees ...*seq.Grammar) (*seq.Grammar, error) {
	if len(trees) == 1 {
		return trees[0], nil
	}
	first := map[string]string{}
	for _, g := range trees {
		if _, ok := first[g.Name]; ok {
			return nil, fmt.Errorf("two trees are called %s", g.Name)
		}
		first[g.Name] = g.Name + "." + g.Scopes[0]
	}

	out := &seq.Grammar{Name: "trees", Params: map[string]string{}, Delays: map[string]time.Duration{}, Settles: map[string]string{}}
	for _, g := range trees {
		local := map[string]bool{}
		for _, s := range g.Scopes {
			local[s] = true
		}
		at := func(scope string, line int) (string, error) {
			switch {
			case scope == "":
				return "", nil
			case local[scope]:
				return g.Name + "." + scope, nil
			case first[scope] != "":
				return first[scope], nil
			}
			return "", fmt.Errorf("%s:%d: -> %s, and there is no such scope or tree", g.Name, line, scope)
		}
		for _, s := range g.Scopes {
			name, _ := at(s, 0)
			out.Scopes = append(out.Scopes, name)
			if p := g.Params[s]; p != "" {
				out.Params[name] = p
			}
			if d, ok := g.Settles[s]; ok {
				out.Settles[name] = d
			}
		}
		for _, name := range slices.Sorted(maps.Keys(g.Delays)) {
			d := g.Delays[name]
			if was, ok := out.Delays[name]; ok && was != d {
				return nil, fmt.Errorf("%s waits %s for %s, and another tree %s", g.Name, d, name, was)
			}
			out.Delays[name] = d
		}
		named := func(scope, name string, line int) (string, error) {
			if p := g.Params[scope]; p != "" && name == p {
				return name, nil
			}
			return at(name, line)
		}
		var err error
		for _, r := range g.Rules {
			home := r.Scope
			if r.Scope, err = at(home, r.Line); err != nil {
				return nil, err
			}
			if r.Goto, err = named(home, r.Goto, r.Line); err != nil {
				return nil, err
			}
			if r.Arg, err = named(home, r.Arg, r.Line); err != nil {
				return nil, err
			}
			if r.Include, err = at(r.Include, r.Line); err != nil {
				return nil, err
			}
			out.Rules = append(out.Rules, r)
		}
		for _, x := range g.Any {
			scopes := []string{x.Scope}
			if x.Scope == "" {
				scopes = g.Scopes
			}
			if x.Goto, err = named(x.Scope, x.Goto, x.Line); err != nil {
				return nil, err
			}
			if x.Arg, err = named(x.Scope, x.Arg, x.Line); err != nil {
				return nil, err
			}
			for _, s := range scopes {
				x.Scope, _ = at(s, x.Line)
				out.Any = append(out.Any, x)
			}
		}
	}
	return out, nil
}

func Compile(trees ...*seq.Grammar) (*seq.Table, error) {
	g, err := Link(trees...)
	if err != nil {
		return nil, err
	}
	return seq.Compile(g, Alphabet(g))
}
