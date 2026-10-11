package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/cptaffe/automaton/lex/draw"
	"github.com/cptaffe/automaton/lex/gen"
)

func main() { os.Exit(run(os.Args[1:], os.Stderr)) }

func run(args []string, stderr io.Writer) int {
	fs := flag.NewFlagSet("genlex", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dir := fs.String("dot", "", "write Graphviz drawings into this directory")
	goOut := fs.String("go", "", "write the table here, as Go")
	pkg := fs.String("package", "", "the package the Go table belongs to")
	verbose := fs.Bool("v", false, "describe each scope's machine")
	fs.Usage = func() {
		fmt.Fprintln(stderr, "usage: genlex [-v] [-dot dir] [-go out.go -package name] file.lex...")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return 2
	}

	var err error
	switch {
	case fs.NArg() == 0:
		fs.Usage()
		return 2
	case *goOut != "" && fs.NArg() > 1:
		err = errors.New("-go writes one table, so it takes one file")
	case *goOut != "" && *pkg == "":
		err = errors.New("-go needs -package")
	}
	for _, path := range fs.Args() {
		if err != nil {
			break
		}
		err = one(path, *dir, *goOut, *pkg, *verbose, stderr)
	}
	if err != nil {
		fmt.Fprintln(stderr, "genlex:", err)
		return 1
	}
	return 0
}

func one(path, dir, goOut, pkg string, verbose bool, stderr io.Writer) error {
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
		fmt.Fprintf(stderr, "genlex: shadowed: %s\n", s)
	}
	for _, m := range ms {
		if verbose {
			fmt.Fprintf(stderr, "genlex: %s\n", draw.Summary(m))
		}
		for _, c := range m.Covered {
			fmt.Fprintf(stderr, "genlex: %s\n", c)
		}
	}
	if len(dead) > 0 {
		return errors.New(strings.Join(dead, "\n"))
	}

	if dir != "" {
		shape, err := draw.Shape(flat)
		if err != nil {
			return err
		}
		whole, err := draw.Whole(flat, ms)
		if err != nil {
			return err
		}
		if err := write(filepath.Join(dir, flat.Name+"-shape.dot"), shape); err != nil {
			return err
		}
		if err := write(filepath.Join(dir, flat.Name+".dot"), whole); err != nil {
			return err
		}
	}

	if goOut == "" {
		return nil
	}
	spec, err := gen.Spec(flat, ms)
	if err != nil {
		return err
	}
	src, err := gen.Go(spec, pkg, filepath.Base(path))
	if err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	return write(goOut, src)
}

func write(path string, b []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}
