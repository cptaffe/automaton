package lexgen

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

		if s.Includes != "" {
			return nil, fmt.Errorf("%s:%d: %s takes a parameter and includes %s; a template cannot include",
				f.Name, s.Line, s.Name, s.Includes)
		}
	}

	if entry == "" {
		return nil, fmt.Errorf("%s: every scope takes a parameter, so there is nowhere to begin", f.Name)
	}

	out := &Flat{Name: f.Name, Shadowed: f.Shadowed, Defs: f.Defs}
	done := map[string]bool{}

	var want func(name, arg string, chain []string) (string, error)

	want = func(name, arg string, chain []string) (string, error) {
		s := f.Scope(name)

		if s == nil {
			return "", fmt.Errorf("%s: -> %s, and there is no such scope", f.Name, name)
		}

		if s.Param == "" {
			if arg != "" {
				return "", fmt.Errorf("%s:%d: %s takes no parameter, and %s was given one", f.Name, s.Line, name, arg)
			}

			at := name

			if done[at] {
				return at, nil
			}

			done[at] = true

			return at, emit(f, out, s, at, "", "", want, chain)
		}

		if arg == "" {
			return "", fmt.Errorf("%s:%d: %s takes a parameter and was used without one", f.Name, s.Line, name)
		}

		at := name + "(" + arg + ")"

		if done[at] {
			return at, nil
		}

		for _, was := range chain {
			if was == name {
				return "", fmt.Errorf("%s:%d: %s stands inside itself, so where it goes next is not known until it runs; that needs a stack, which is not built",
					f.Name, s.Line, name)
			}
		}

		done[at] = true

		return at, emit(f, out, s, at, s.Param, arg, want, append(chain, name))
	}

	if _, err := want(entry, "", nil); err != nil {
		return nil, err
	}

	sort.SliceStable(out.Scopes, func(i, j int) bool { return out.Scopes[i].Line < out.Scopes[j].Line })

	for _, s := range f.Scopes {
		used := false

		for _, got := range out.Scopes {
			if got.Name == s.Name || strings.HasPrefix(got.Name, s.Name+"(") {
				used = true
			}
		}

		if !used {
			return nil, fmt.Errorf("%s:%d: nothing reaches %s", f.Name, s.Line, s.Name)
		}
	}

	return out, nil
}

type wanter func(name, arg string, chain []string) (string, error)

func emit(f *File, out *Flat, s *Scope, at, param, arg string, want wanter, chain []string) error {
	out.Scopes = append(out.Scopes, FlatScope{Name: at, Includes: s.Includes, Line: s.Line})

	if s.Includes != "" {
		if _, err := want(s.Includes, "", chain); err != nil {
			return err
		}
	}

	for _, r := range f.Rules {
		if r.Scope != s.Name {
			continue
		}

		got := FlatRule{Expr: r.Expr, Scope: at, Line: r.Line, Text: r.Text,
			Acts: FlatActs{Emit: r.Acts.Emit, Do: r.Acts.Do, Goto: r.Acts.Goto, Drop: r.Acts.Drop}}

		if r.Acts.Goto != "" {
			to, a := r.Acts.Goto, r.Acts.Arg

			if to == param {
				if a != "" {
					return fmt.Errorf("%s:%d: %s is this scope's parameter and cannot take one of its own", f.Name, r.Line, to)
				}

				to, a = arg, ""
			}

			if a == "this" {
				a = at
			}

			to, err := want(to, a, chain)

			if err != nil {
				return err
			}

			got.Acts.Goto = to
		}

		out.Rules = append(out.Rules, got)
	}

	return nil
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
