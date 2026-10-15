package gen

import (
	"cmp"
	"fmt"
	"slices"
	"strings"
)

func Expand(f *File) (*Flat, error) {
	if err := cycles(f); err != nil {
		return nil, err
	}

	entry := f.entry()

	if entry == "" {
		return nil, fmt.Errorf("%s: every scope takes a parameter, so there is nowhere to begin", f.Path)
	}

	e := &expander{
		file: f,
		out:  &Flat{Path: f.Path, Shadowed: f.Shadowed, Defs: f.Defs},
		done: map[string]bool{},
		dyn:  dynamic(f),
	}

	if _, err := e.want(entry, "", nil); err != nil {
		return nil, err
	}

	slices.SortStableFunc(e.out.Scopes, func(a, b FlatScope) int { return cmp.Compare(a.Line, b.Line) })

	if err := e.unreached(); err != nil {
		return nil, err
	}

	return e.out, nil
}

func (f *File) entry() string {
	for _, s := range f.Scopes {
		if s.Param == "" {
			return s.Name
		}
	}

	return ""
}

type expander struct {
	file *File
	out  *Flat
	done map[string]bool
	dyn  map[string]bool
}

type instance struct {
	at    string
	param string
	arg   string
	chain []string
}

func (in instance) runtime() bool { return in.param != "" && in.arg == "" }

func (e *expander) want(name, arg string, chain []string) (string, error) {
	f := e.file
	s := f.Scope(name)

	switch {
	case s == nil:
		return "", fmt.Errorf("%s: -> %s, and there is no such scope", f.Path, name)
	case s.Param == "" && arg != "":
		return "", fmt.Errorf("%s:%d: %s takes no parameter, and %s was given one", f.Path, s.Line, name, arg)
	case s.Param == "":
		return e.instantiate(s, instance{at: name, chain: chain})
	case arg == "":
		return "", fmt.Errorf("%s:%d: %s takes a parameter and was used without one", f.Path, s.Line, name)
	case e.dyn[name]:
		return e.instantiate(s, instance{at: name, param: s.Param, chain: chain})
	}

	at := name + "(" + arg + ")"

	if e.done[at] {
		return at, nil
	}

	if slices.Contains(chain, name) {
		return "", fmt.Errorf("%s:%d: %s stands inside itself, so where it goes next is not known until it runs; that needs a stack, which is not built", f.Path, s.Line, name)
	}

	return e.instantiate(s, instance{at: at, param: s.Param, arg: arg, chain: append(chain, name)})
}

func (e *expander) instantiate(s *Scope, in instance) (string, error) {
	if e.done[in.at] {
		return in.at, nil
	}

	e.done[in.at] = true
	e.out.Scopes = append(e.out.Scopes, FlatScope{Name: in.at, Line: s.Line})

	return in.at, e.rules(s.Name, in, false)
}

func (e *expander) rules(scope string, in instance, inlined bool) error {
	for _, r := range e.file.Rules {
		if r.Scope != scope {
			continue
		}

		if r.Include != "" {
			if !inlined {
				e.out.Rules = append(e.out.Rules, FlatRule{Include: r.Include, Scope: in.at, Line: r.Line, Text: r.Text})
			}

			if err := e.rules(r.Include, instance{at: in.at, chain: in.chain}, true); err != nil {
				return err
			}

			continue
		}

		got := FlatRule{Expr: r.Expr, Scope: in.at, Line: r.Line, Text: r.Text, Inlined: inlined,
			Acts: FlatActs{Emit: r.Acts.Emit, Do: r.Acts.Do, Drop: r.Acts.Drop}}

		if r.Acts.Goto != "" {
			acts, err := e.move(r, in)

			if err != nil {
				return err
			}

			got.Acts.Goto, got.Acts.Push, got.Acts.Pop = acts.Goto, acts.Push, acts.Pop
		}

		e.out.Rules = append(e.out.Rules, got)
	}

	return nil
}

func (e *expander) move(r Rule, in instance) (FlatActs, error) {
	to, a := r.Acts.Goto, r.Acts.Arg

	if to == in.param {
		if a != "" {
			return FlatActs{}, fmt.Errorf("%s:%d: %s is this scope's parameter and cannot take one of its own", e.file.Path, r.Line, to)
		}

		if in.runtime() {
			return FlatActs{Pop: true}, nil
		}

		to, a = in.arg, ""
	}

	if a == in.param && in.param != "" && !in.runtime() {
		a = in.arg
	}

	if !e.dyn[to] {
		if a == "this" {
			a = in.at
		}

		got, err := e.want(to, a, in.chain)

		return FlatActs{Goto: got}, err
	}

	if a == "" {
		return FlatActs{}, fmt.Errorf("%s:%d: %s takes a parameter and was used without one", e.file.Path, r.Line, to)
	}

	got, err := e.want(to, a, in.chain)

	if err != nil {
		return FlatActs{}, err
	}

	switch {
	case a == "this":
		return FlatActs{Goto: got, Push: in.at}, nil
	case in.runtime() && a == in.param:
		return FlatActs{Goto: got}, nil
	}

	back, err := e.want(a, "", in.chain)

	return FlatActs{Goto: got, Push: back}, err
}

func (e *expander) unreached() error {
	included := map[string]bool{}

	for _, r := range e.file.Rules {
		if r.Include != "" {
			included[r.Include] = true
		}
	}

	for _, s := range e.file.Scopes {
		if !included[s.Name] && !e.out.instantiates(s.Name) {
			return fmt.Errorf("%s:%d: nothing reaches %s", e.file.Path, s.Line, s.Name)
		}
	}

	return nil
}

func (f *Flat) instantiates(scope string) bool {
	for _, s := range f.Scopes {
		if s.Name == scope || strings.HasPrefix(s.Name, scope+"(") {
			return true
		}
	}

	return false
}

func cycles(f *File) error {
	into := map[string][]Rule{}

	for _, r := range f.Rules {
		if r.Include != "" {
			into[r.Scope] = append(into[r.Scope], r)
		}
	}

	state := map[string]int{}

	var visit func(at string) error

	visit = func(at string) error {
		state[at] = 1

		for _, r := range into[at] {
			switch state[r.Include] {
			case 1:
				return fmt.Errorf("%s:%d: -> %s goes round in a loop back to %s", f.Path, r.Line, r.Include, at)
			case 0:
				if err := visit(r.Include); err != nil {
					return err
				}
			}
		}

		state[at] = 2

		return nil
	}

	for _, s := range f.Scopes {
		if state[s.Name] == 0 {
			if err := visit(s.Name); err != nil {
				return err
			}
		}
	}

	return nil
}

func dynamic(f *File) map[string]bool {
	g := callsOf(f)
	dyn := map[string]bool{}

	var work []string

	for name := range g.param {
		if g.reaches(name, name) {
			dyn[name] = true
			work = append(work, name)
		}
	}

	for len(work) > 0 {
		at := work[len(work)-1]
		work = work[:len(work)-1]

		for _, c := range g.calls[at] {
			if c.arg == g.param[at] && !dyn[c.to] {
				dyn[c.to] = true
				work = append(work, c.to)
			}
		}
	}

	return dyn
}

type call struct {
	to  string
	arg string
}

type callGraph struct {
	param map[string]string
	calls map[string][]call
}

func callsOf(f *File) callGraph {
	g := callGraph{param: map[string]string{}, calls: map[string][]call{}}

	for _, s := range f.Scopes {
		if s.Param != "" {
			g.param[s.Name] = s.Param
		}
	}

	for _, r := range f.Rules {
		_, from := g.param[r.Scope]
		_, to := g.param[r.Acts.Goto]

		if from && to && r.Acts.Goto != g.param[r.Scope] {
			g.calls[r.Scope] = append(g.calls[r.Scope], call{to: r.Acts.Goto, arg: r.Acts.Arg})
		}
	}

	return g
}

func (g callGraph) reaches(from, to string) bool {
	seen := map[string]bool{}
	work := []string{from}

	for len(work) > 0 {
		at := work[len(work)-1]
		work = work[:len(work)-1]

		for _, c := range g.calls[at] {
			if c.to == to {
				return true
			}

			if !seen[c.to] {
				seen[c.to] = true
				work = append(work, c.to)
			}
		}
	}

	return false
}

func (f *Flat) Instantiations() []string {
	var out []string

	for _, s := range f.Scopes {
		if strings.Contains(s.Name, "(") {
			out = append(out, s.Name)
		}
	}

	return out
}
