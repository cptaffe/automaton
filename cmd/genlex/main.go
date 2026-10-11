package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cptaffe/automaton/lex/draw"
	"github.com/cptaffe/automaton/lex/gen"
)

func main() {
	dir := flag.String("dot", "", "write Graphviz into this directory")
	goOut := flag.String("go", "", "write the table here, as Go")
	pkg := flag.String("package", "", "the package the table belongs to")
	flag.Parse()

	for _, path := range flag.Args() {
		if err := run(path, *dir, *goOut, *pkg); err != nil {
			fmt.Fprintln(os.Stderr, "genlex:", err)
			os.Exit(1)
		}
	}
}

func run(path, dir, goOut, pkg string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	parsed, err := gen.Parse(f, path)
	f.Close()
	if err != nil {
		return err
	}
	flat, err := gen.Expand(parsed)
	if err != nil {
		return err
	}
	ms, dead, err := gen.Build(flat)
	if err != nil {
		return err
	}

	for _, s := range flat.Shadowed {
		fmt.Printf("genlex:   shadowed: %s\n", s)
	}
	for _, m := range ms {
		fmt.Printf("genlex: %s\n", draw.Summary(m))
		for _, c := range m.Covered {
			fmt.Printf("genlex:   %s\n", c)
		}
	}

	if dir != "" {
		if err := write(filepath.Join(dir, flat.Name+"-shape.dot"), draw.Shape(flat)); err != nil {
			return err
		}
		if err := write(filepath.Join(dir, flat.Name+".dot"), draw.Whole(flat, ms)); err != nil {
			return err
		}
	}

	if len(dead) > 0 {
		return fmt.Errorf("%s", dead[0])
	}
	if goOut == "" {
		return nil
	}
	spec, bad := gen.Spec(flat, ms)
	if len(bad) > 0 {
		return fmt.Errorf("%s", strings.Join(bad, "; "))
	}
	return write(goOut, gen.Go(flat, spec, pkg))
}

func write(path string, b []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(path, b, 0o644); err != nil {
		return err
	}
	fmt.Printf("genlex:   %s\n", path)
	return nil
}
