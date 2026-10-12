package gen

import (
	"fmt"
	"sort"
	"strings"
)

func Expand(f *File) (*Flat, error) {
	var entry string

	for _, s := range f.Scopes {
		if s.Param == "" {
			entry = s.Name

			break
		}
	}

	if err := cycles(f); err != nil {
		return nil, err
	}

	if entry == "" {
		return nil, fmt.Errorf("%s: every scope takes a parameter, so there is nowhere to begin", f.Path)
	}

	out := &Flat{Path: f.Path, Shadowed: f.Shadowed, Defs: f.Defs}
	done := map[string]bool{}
	dyn := dynamic(f)

	var want func(name, arg string, chain []string) (string, error)

	want = func(name, arg string, chain []string) (string, error) {
		s := f.Scope(name)

		if s == nil {
			return "", fmt.Errorf("%s: -> %s, and there is no such scope", f.Path, name)
		}

		if s.Param == "" {
			if arg != "" {
				return "", fmt.Errorf("%s:%d: %s takes no parameter, and %s was given one", f.Path, s.Line, name, arg)
			}

			at := name

			if done[at] {
				return at, nil
			}

			done[at] = true

			return at, emit(f, out, s, at, "", "", dyn, want, chain)
		}

		if arg == "" {
			return "", fmt.Errorf("%s:%d: %s takes a parameter and was used without one", f.Path, s.Line, name)
		}

		if dyn[name] {
			if done[name] {
				return name, nil
			}

			done[name] = true

			return name, emit(f, out, s, name, s.Param, "", dyn, want, chain)
		}

		at := name + "(" + arg + ")"

		if done[at] {
			return at, nil
		}

		for _, was := range chain {
			if was == name {
				return "", fmt.Errorf("%s:%d: %s stands inside itself, so where it goes next is not known until it runs; that needs a stack, which is not built", f.Path, s.Line, name)
			}
		}

		done[at] = true

		return at, emit(f, out, s, at, s.Param, arg, dyn, want, append(chain, name))
	}

	if _, err := want(entry, "", nil); err != nil {
		return nil, err
	}

	sort.SliceStable(out.Scopes, func(i, j int) bool { return out.Scopes[i].Line < out.Scopes[j].Line })

	included := map[string]bool{}

	for _, r := range f.Rules {
		if r.Include != "" {
			included[r.Include] = true
		}
	}

	for _, s := range f.Scopes {
		used := included[s.Name]

		for _, got := range out.Scopes {
			if got.Name == s.Name || strings.HasPrefix(got.Name, s.Name+"(") {
				used = true
			}
		}

		if !used {
			return nil, fmt.Errorf("%s:%d: nothing reaches %s", f.Path, s.Line, s.Name)
		}
	}

	return out, nil
}

type wanter func(name, arg string, chain []string) (string, error)

func emit(f *File, out *Flat, s *Scope, at, param, arg string, dyn map[string]bool, want wanter, chain []string) error {
	out.Scopes = append(out.Scopes, FlatScope{Name: at, Line: s.Line})

	return rules(f, out, s.Name, at, param, arg, false, dyn, want, chain)
}

func rules(f *File, out *Flat, scope, at, param, arg string, inlined bool, dyn map[string]bool, want wanter, chain []string) error {
	for _, r := range f.Rules {
		if r.Scope != scope {
			continue
		}

		if r.Include != "" {
			if !inlined {
				out.Rules = append(out.Rules, FlatRule{Include: r.Include, Scope: at, Line: r.Line, Text: r.Text})
			}

			if err := rules(f, out, r.Include, at, "", "", true, dyn, want, chain); err != nil {
				return err
			}

			continue
		}

		got := FlatRule{Expr: r.Expr, Scope: at, Line: r.Line, Text: r.Text, Inlined: inlined,
			Acts: FlatActs{Emit: r.Acts.Emit, Do: r.Acts.Do, Drop: r.Acts.Drop}}

		if r.Acts.Goto != "" {
			acts, err := move(f, r, at, param, arg, dyn, want, chain)

			if err != nil {
				return err
			}

			got.Acts.Goto, got.Acts.Push, got.Acts.Pop = acts.Goto, acts.Push, acts.Pop
		}

		out.Rules = append(out.Rules, got)
	}

	return nil
}

func move(f *File, r Rule, at, param, arg string, dyn map[string]bool, want wanter, chain []string) (FlatActs, error) {
	to, a := r.Acts.Goto, r.Acts.Arg
	inDyn := param != "" && arg == ""

	if to == param {
		if a != "" {
			return FlatActs{}, fmt.Errorf("%s:%d: %s is this scope's parameter and cannot take one of its own", f.Path, r.Line, to)
		}

		if inDyn {
			return FlatActs{Pop: true}, nil
		}

		to, a = arg, ""
	}

	if a == param && param != "" && !inDyn {
		a = arg
	}

	if !dyn[to] {
		if a == "this" {
			a = at
		}

		got, err := want(to, a, chain)

		return FlatActs{Goto: got}, err
	}

	if a == "" {
		return FlatActs{}, fmt.Errorf("%s:%d: %s takes a parameter and was used without one", f.Path, r.Line, to)
	}

	got, err := want(to, a, chain)

	if err != nil {
		return FlatActs{}, err
	}

	switch {
	case a == "this":
		return FlatActs{Goto: got, Push: at}, nil
	case inDyn && a == param:
		return FlatActs{Goto: got}, nil
	}

	back, err := want(a, "", chain)

	return FlatActs{Goto: got, Push: back}, err
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
	param := map[string]string{}

	for _, s := range f.Scopes {
		if s.Param != "" {
			param[s.Name] = s.Param
		}
	}

	calls := map[string][]Rule{}

	for _, r := range f.Rules {
		if _, ok := param[r.Scope]; ok && r.Acts.Goto != "" && r.Acts.Goto != param[r.Scope] {
			calls[r.Scope] = append(calls[r.Scope], r)
		}
	}

	reaches := func(from, to string) bool {
		seen := map[string]bool{}
		work := []string{from}

		for len(work) > 0 {
			at := work[len(work)-1]
			work = work[:len(work)-1]

			for _, r := range calls[at] {
				next := r.Acts.Goto

				if _, ok := param[next]; !ok || seen[next] {
					continue
				}

				if next == to {
					return true
				}

				seen[next] = true
				work = append(work, next)
			}
		}

		return false
	}

	dyn := map[string]bool{}

	for name := range param {
		dyn[name] = reaches(name, name)
	}

	for changed := true; changed; {
		changed = false

		for name := range param {
			if !dyn[name] {
				continue
			}

			for _, r := range calls[name] {
				if _, ok := param[r.Acts.Goto]; ok && r.Acts.Arg == param[name] && !dyn[r.Acts.Goto] {
					dyn[r.Acts.Goto], changed = true, true
				}
			}
		}
	}

	return dyn
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
