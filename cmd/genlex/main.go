package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cptaffe/automaton"
	"github.com/cptaffe/automaton/dot"
	"github.com/cptaffe/automaton/lexgen"
)

type drawn struct {
	m     lexgen.Machine
	names []lexgen.Def
}

func (d drawn) Named(s automaton.Set) (string, bool) {
	best := ""

	for _, def := range d.names {
		got, ok := def.Expr.Single()

		if !ok || got.Empty() {
			continue
		}

		if got.Equal(s) {
			return def.Name, true
		}

		if !s.Without(got).Empty() {
			continue
		}

		say := def.Name + " - " + d.name(got.Without(s))

		if best == "" || len(say) < len(best) {
			best = say
		}
	}

	if best == "" || len(best) > 24 || len(best) >= len(s.Spelled()) {
		return "", false
	}

	return best, true
}

func (d drawn) name(s automaton.Set) string {
	for _, def := range d.names {
		if got, ok := def.Expr.Single(); ok && got.Equal(s) {
			return def.Name
		}
	}

	return "[" + s.Spelled() + "]"
}

func (d drawn) Leaves(s automaton.State) (string, bool) {
	a := d.m.DFA.Accept(s)

	if a == automaton.NoMatch {
		return "", false
	}

	to := d.m.Rules[a].Acts.Goto

	return to, to != "" && to != d.m.Scope
}

func (d drawn) Name() string { return d.m.Scope }

func (d drawn) Note() string {
	return fmt.Sprintf("%s -- %s, %s, %s", d.m.Scope,
		plural(len(d.m.Rules), "rule"), plural(d.m.DFA.States(), "state"),
		plural(d.m.DFA.Classes().Len(), "class"))
}

func (d drawn) States() int { return d.m.DFA.States() }

func (d drawn) Starts() map[string]automaton.State {
	return map[string]automaton.State{d.m.Scope: d.m.DFA.Start()}
}

func (d drawn) Classes() *automaton.Classes { return d.m.DFA.Classes() }

func (d drawn) Next(s automaton.State, c automaton.Class) automaton.State {
	return d.m.DFA.Next(s, c)
}

func (d drawn) Accepts(s automaton.State) (string, bool) {
	a := d.m.DFA.Accept(s)

	if a == automaton.NoMatch {
		return "", false
	}

	r := d.m.Rules[a]

	switch {
	case r.Acts.Emit != "":
		return r.Acts.Emit, true
	case r.Acts.Drop:
		return "ignore", true
	}

	return "-> " + r.Acts.Goto, true
}

func plural(n int, what string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, what)
	}

	if strings.HasSuffix(what, "s") {
		return fmt.Sprintf("%d %ses", n, what)
	}

	return fmt.Sprintf("%d %ss", n, what)
}

func main() {
	out := flag.String("dot", "", "write Graphviz here")
	gen := flag.String("go", "", "write the table here, as Go")
	pkg := flag.String("package", "", "the package the table belongs to")
	flag.Parse()

	if err := run(flag.Args(), *out, *gen, *pkg); err != nil {
		fmt.Fprintln(os.Stderr, "genlex:", err)
		os.Exit(1)
	}
}

func run(paths []string, dir, gen, pkg string) error {
	for _, path := range paths {
		f, err := os.Open(path)

		if err != nil {
			return err
		}

		got, err := lexgen.Parse(f, path)
		f.Close()

		if err != nil {
			return err
		}

		flat, err := lexgen.Expand(got)

		if err != nil {
			return err
		}

		ms, dead, err := lexgen.Build(flat)

		if err != nil {
			return err
		}

		for _, s := range flat.Shadowed {
			fmt.Printf("genlex:   shadowed: %s\n", s)
		}

		for _, m := range ms {
			fmt.Printf("genlex: %s, %s, %s, %s\n", m.Scope,
				plural(len(m.Rules), "rule"), plural(m.DFA.States(), "state"),
				plural(m.DFA.Classes().Len(), "class"))

			for _, c := range m.Covered {
				fmt.Printf("genlex:   %s\n", c)
			}

			if dir == "" {
				continue
			}

			if err := os.MkdirAll(dir, 0o755); err != nil {
				return err
			}

			_ = m
		}

		if dir != "" {
			shape := filepath.Join(dir, flat.Name+"-shape.dot")

			if err := os.WriteFile(shape, shapeOf(flat), 0o644); err != nil {
				return err
			}

			fmt.Printf("genlex:   %s\n", shape)

			var all []dot.Machine

			for _, m := range ms {
				all = append(all, drawn{m: m, names: flat.Defs})
			}

			at := filepath.Join(dir, flat.Name+".dot")

			if err := os.WriteFile(at, dot.Whole(all, flat.Name), 0o644); err != nil {
				return err
			}

			fmt.Printf("genlex:   %s\n", at)
		}

		for _, d := range dead {
			return fmt.Errorf("%s", d)
		}

		if gen == "" {
			continue
		}

		spec, bad := lexgen.Spec(flat, ms)

		if len(bad) != 0 {
			return fmt.Errorf("%s", strings.Join(bad, "; "))
		}

		if err := os.MkdirAll(filepath.Dir(gen), 0o755); err != nil {
			return err
		}

		if err := os.WriteFile(gen, lexgen.Go(flat, spec, pkg), 0o644); err != nil {
			return err
		}

		fmt.Printf("genlex:   %s\n", gen)
	}

	return nil
}

func safe(s string) string {
	out := []rune{}

	for _, r := range s {
		if r == '(' || r == ')' {
			r = '-'
		}

		out = append(out, r)
	}

	return string(out)
}
