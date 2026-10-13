package seq

import (
	"fmt"
	"strings"
)

func expand(g *Grammar) (*Grammar, error) {
	if len(g.Params) == 0 {
		return g, nil
	}
	out := &Grammar{Name: g.Name, Note: g.Note, Delays: g.Delays}
	done := map[string]bool{}

	var want func(name, arg string) (string, error)
	emit := func(scope, at, param, arg string) error {
		out.Scopes = append(out.Scopes, at)
		move := func(to, a string) (string, error) {
			switch {
			case to == "":
				return "", nil
			case param != "" && to == param:
				return arg, nil
			case param != "" && a == param:
				a = arg
			case a == "this":
				a = at
			}
			return want(to, a)
		}
		for _, r := range g.Rules {
			if r.Scope != scope {
				continue
			}
			to, err := move(r.Goto, r.Arg)
			if err != nil {
				return err
			}
			if r.Include != "" {
				if _, err := want(r.Include, ""); err != nil {
					return err
				}
			}
			r.Scope, r.Goto, r.Arg = at, to, ""
			out.Rules = append(out.Rules, r)
		}
		for _, x := range g.Any {
			if x.Scope != scope && !(x.Scope == "" && scope == g.Scopes[0]) {
				continue
			}
			to, err := move(x.Goto, x.Arg)
			if err != nil {
				return err
			}
			if x.Scope != "" {
				x.Scope = at
			}
			x.Goto, x.Arg = to, ""
			out.Any = append(out.Any, x)
		}
		return nil
	}
	want = func(name, arg string) (string, error) {
		param := g.Params[name]
		at := name
		if param != "" {
			if strings.Contains("("+arg, "("+name+"(") {
				return "", fmt.Errorf("%s: %s(%s) holds %s inside itself, and a scope has no stack to come back by", g.Name, name, arg, name)
			}
			at = name + "(" + arg + ")"
		}
		if done[at] {
			return at, nil
		}
		done[at] = true
		return at, emit(name, at, param, arg)
	}
	for _, s := range g.Scopes {
		if g.Params[s] == "" {
			if _, err := want(s, ""); err != nil {
				return nil, err
			}
		}
	}
	return out, nil
}
