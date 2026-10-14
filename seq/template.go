package seq

import (
	"fmt"
	"strings"
)

type instance struct {
	scope string
	at    string
	param string
	arg   string
}

type expander struct {
	g    *Grammar
	out  *Grammar
	done map[string]bool
}

func expand(g *Grammar) (*Grammar, error) {
	if len(g.Params) == 0 {
		return g, nil
	}
	e := &expander{g: g, out: &Grammar{Name: g.Name, Note: g.Note, Delays: g.Delays}, done: map[string]bool{}}
	for _, s := range g.Scopes {
		if g.Params[s] == "" {
			if _, err := e.want(s, ""); err != nil {
				return nil, err
			}
		}
	}
	return e.out, nil
}

func (e *expander) want(name, arg string) (string, error) {
	in := instance{scope: name, at: name, param: e.g.Params[name], arg: arg}
	if in.param != "" {
		if strings.Contains("("+arg, "("+name+"(") {
			return "", fmt.Errorf("%s: %s(%s) holds %s inside itself, and a scope has no stack to come back by", e.g.Name, name, arg, name)
		}
		in.at = name + "(" + arg + ")"
	}
	if e.done[in.at] {
		return in.at, nil
	}
	e.done[in.at] = true
	return in.at, e.emit(in)
}

func (e *expander) emit(in instance) error {
	e.out.Scopes = append(e.out.Scopes, in.at)
	for _, r := range e.g.Rules {
		if r.Scope != in.scope {
			continue
		}
		if err := e.rule(in, r); err != nil {
			return err
		}
	}
	for _, x := range e.g.Any {
		if !e.reads(in, x) {
			continue
		}
		if err := e.wildcard(in, x); err != nil {
			return err
		}
	}
	return nil
}

func (e *expander) reads(in instance, x Any) bool {
	return x.Scope == in.scope || x.Scope == "" && in.scope == e.g.Scopes[0]
}

func (e *expander) rule(in instance, r Rule) error {
	to, err := e.move(in, r.move())
	if err != nil {
		return err
	}
	r.Scope, r.Goto, r.Arg = in.at, to, ""
	e.out.Rules = append(e.out.Rules, r)
	return nil
}

func (e *expander) wildcard(in instance, x Any) error {
	to, err := e.move(in, x.move())
	if err != nil {
		return err
	}
	if x.Scope != "" {
		x.Scope = in.at
	}
	x.Goto, x.Arg = to, ""
	e.out.Any = append(e.out.Any, x)
	return nil
}

func (e *expander) move(in instance, m move) (string, error) {
	switch {
	case m.to == "":
		return "", nil
	case in.param != "" && m.to == in.param:
		return in.arg, nil
	case in.param != "" && m.arg == in.param:
		m.arg = in.arg
	case m.arg == "this":
		m.arg = in.at
	}
	return e.want(m.to, m.arg)
}
