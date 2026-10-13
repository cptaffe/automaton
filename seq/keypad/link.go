package keypad

import (
	"fmt"
	"maps"
	"slices"
	"time"

	"github.com/cptaffe/automaton/seq"
)

type linker struct {
	out   *seq.Grammar
	first map[string]string
}

type renamer struct {
	tree  *seq.Grammar
	local map[string]bool
	first map[string]string
}

func Link(trees ...*seq.Grammar) (*seq.Grammar, error) {
	if len(trees) == 1 {
		return trees[0], nil
	}
	l := &linker{
		out:   &seq.Grammar{Name: "trees", Params: map[string]string{}, Delays: map[string]time.Duration{}},
		first: map[string]string{},
	}
	for _, g := range trees {
		if _, ok := l.first[g.Name]; ok {
			return nil, fmt.Errorf("two trees are called %s", g.Name)
		}
		l.first[g.Name] = g.Name + "." + g.Scopes[0]
	}

	for _, g := range trees {
		if err := l.link(g); err != nil {
			return nil, err
		}
	}
	return l.out, nil
}

func (l *linker) link(g *seq.Grammar) error {
	rn := renamer{tree: g, local: map[string]bool{}, first: l.first}
	for _, s := range g.Scopes {
		rn.local[s] = true
	}
	for _, s := range g.Scopes {
		name, _ := rn.at(s, 0)
		l.out.Scopes = append(l.out.Scopes, name)
		if p := g.Params[s]; p != "" {
			l.out.Params[name] = p
		}
	}
	if err := l.delays(g); err != nil {
		return err
	}
	for _, r := range g.Rules {
		r, err := rn.rule(r)
		if err != nil {
			return err
		}
		l.out.Rules = append(l.out.Rules, r)
	}
	for _, x := range g.Any {
		xs, err := rn.wildcard(x)
		if err != nil {
			return err
		}
		l.out.Any = append(l.out.Any, xs...)
	}
	return nil
}

func (l *linker) delays(g *seq.Grammar) error {
	for _, name := range slices.Sorted(maps.Keys(g.Delays)) {
		d := g.Delays[name]
		if was, ok := l.out.Delays[name]; ok && was != d {
			return fmt.Errorf("%s waits %s for %s, and another tree %s", g.Name, d, name, was)
		}
		l.out.Delays[name] = d
	}
	return nil
}

func (rn renamer) at(scope string, line int) (string, error) {
	switch {
	case scope == "":
		return "", nil
	case rn.local[scope]:
		return rn.tree.Name + "." + scope, nil
	case rn.first[scope] != "":
		return rn.first[scope], nil
	}
	return "", fmt.Errorf("%s:%d: -> %s, and there is no such scope or tree", rn.tree.Name, line, scope)
}

func (rn renamer) named(scope, name string, line int) (string, error) {
	if p := rn.tree.Params[scope]; name == "this" || p != "" && name == p {
		return name, nil
	}
	return rn.at(name, line)
}

func (rn renamer) rule(r seq.Rule) (seq.Rule, error) {
	home := r.Scope
	var err error
	if r.Scope, err = rn.at(home, r.Line); err != nil {
		return r, err
	}
	if r.Goto, err = rn.named(home, r.Goto, r.Line); err != nil {
		return r, err
	}
	if r.Arg, err = rn.named(home, r.Arg, r.Line); err != nil {
		return r, err
	}
	if r.Include, err = rn.at(r.Include, r.Line); err != nil {
		return r, err
	}
	return r, nil
}

func (rn renamer) wildcard(x seq.Any) ([]seq.Any, error) {
	var err error
	if x.Goto, err = rn.named(x.Scope, x.Goto, x.Line); err != nil {
		return nil, err
	}
	if x.Arg, err = rn.named(x.Scope, x.Arg, x.Line); err != nil {
		return nil, err
	}
	scopes := []string{x.Scope}
	if x.Scope == "" {
		scopes = rn.tree.Scopes
	}
	var out []seq.Any
	for _, s := range scopes {
		x.Scope, _ = rn.at(s, x.Line)
		out = append(out, x)
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
