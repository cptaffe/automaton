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
	max := fs.Int("max-states", 0, "the most states a scope's machine may have; 0 means 10000")
	fs.Usage = func() {
		fmt.Fprintln(stderr, "usage: genlex [-v] [-max-states n] [-dot dir] [-go out.go -package name] file.lex...")
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
	o := output{dir: *dir, goOut: *goOut, pkg: *pkg, verbose: *verbose, max: *max, stderr: stderr}
	for _, path := range fs.Args() {
		if err != nil {
			break
		}
		err = o.one(path)
	}
	if err != nil {
		fmt.Fprintln(stderr, "genlex:", err)
		return 1
	}
	return 0
}

type output struct {
	dir     string
	goOut   string
	pkg     string
	verbose bool
	max     int
	stderr  io.Writer
}

func (o output) one(path string) error {
	flat, err := expand(path)
	if err != nil {
		return err
	}
	ms, dead, err := gen.Build(flat, o.max)
	if err != nil {
		return err
	}

	o.warn(flat, ms)
	if len(dead) > 0 {
		return errors.New(strings.Join(dead, "\n"))
	}

	if err := o.dot(flat, ms); err != nil {
		return err
	}
	return o.table(path, flat, ms)
}

func expand(path string) (*gen.Flat, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	parsed, err := gen.Parse(f, path)
	f.Close()
	if err != nil {
		return nil, err
	}
	return gen.Expand(parsed)
}

func (o output) warn(flat *gen.Flat, ms []gen.Machine) {
	for _, s := range flat.Shadowed {
		fmt.Fprintf(o.stderr, "genlex: shadowed: %s\n", s)
	}
	for _, m := range ms {
		if o.verbose {
			fmt.Fprintf(o.stderr, "genlex: %s\n", draw.Summary(m))
		}
		for _, c := range m.Covered {
			fmt.Fprintf(o.stderr, "genlex: %s\n", c)
		}
	}
}

func (o output) dot(flat *gen.Flat, ms []gen.Machine) error {
	if o.dir == "" {
		return nil
	}
	shape, err := draw.Shape(flat).Bytes()
	if err != nil {
		return err
	}
	whole, err := draw.Whole(flat, ms).Bytes()
	if err != nil {
		return err
	}
	if err := write(filepath.Join(o.dir, flat.Name()+"-shape.dot"), shape); err != nil {
		return err
	}
	return write(filepath.Join(o.dir, flat.Name()+".dot"), whole)
}

func (o output) table(path string, flat *gen.Flat, ms []gen.Machine) error {
	if o.goOut == "" {
		return nil
	}
	spec, err := gen.Spec(flat, ms)
	if err != nil {
		return err
	}
	src, err := gen.Go(spec, o.pkg, filepath.Base(path))
	if err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	return write(o.goOut, src)
}

func write(path string, b []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}
