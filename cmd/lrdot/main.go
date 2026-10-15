package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/cptaffe/automaton/dot"
	"github.com/cptaffe/automaton/lr"
	"github.com/cptaffe/automaton/lr/draw"
)

func main() { os.Exit(run(os.Args[1:], os.Stderr)) }

func run(args []string, stderr io.Writer) int {
	fs := flag.NewFlagSet("lrdot", flag.ContinueOnError)
	fs.SetOutput(stderr)
	name := fs.String("name", "", "the grammar's name in the drawings; the output's base name if empty")
	shapeOnly := fs.Bool("shape-only", false, "draw only the grammar's shape, not its machine over tokens")
	fs.Usage = func() {
		fmt.Fprintln(stderr, "usage: lrdot [-name n] [-shape-only] y.output out.dot")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 2 {
		fs.Usage()
		return 2
	}
	if err := one(fs.Arg(0), fs.Arg(1), *name, *shapeOnly); err != nil {
		fmt.Fprintln(stderr, "lrdot:", err)
		return 1
	}
	return 0
}

func one(in, out, name string, shapeOnly bool) error {
	if name == "" {
		name = strings.TrimSuffix(filepath.Base(out), ".dot")
	}
	f, err := os.Open(in)
	if err != nil {
		return err
	}
	g, err := lr.Read(f)
	f.Close()
	if err != nil {
		return fmt.Errorf("%s: %w", in, err)
	}
	b, err := draw.Shape(g, name).Bytes()
	if err == nil {
		err = os.WriteFile(out, b, 0o644)
	}
	if err != nil {
		return err
	}
	t, err := draw.Tokens(g, name)
	if err != nil {
		return err
	}
	if t == nil || shapeOnly {
		return nil
	}
	b, err = dot.Draw(t).Bytes()
	if err != nil {
		return err
	}
	return os.WriteFile(strings.TrimSuffix(out, ".dot")+"-tokens.dot", b, 0o644)
}
