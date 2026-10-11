package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/cptaffe/automaton/seq"
	seqdraw "github.com/cptaffe/automaton/seq/draw"
	"github.com/cptaffe/automaton/seq/hotkey"
	"github.com/cptaffe/automaton/seq/keypad"
)

func main() { os.Exit(run(os.Args[1:], os.Stderr)) }

func run(args []string, stderr io.Writer) int {
	fs := flag.NewFlagSet("genseq", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dialect := fs.String("d", "", "the dialect, keypad or hotkey; by default .tree is keypad and anything else hotkey")
	dir := fs.String("dot", "", "write Graphviz drawings into this directory")
	verbose := fs.Bool("v", false, "describe each table")
	fs.Usage = func() {
		fmt.Fprintln(stderr, "usage: genseq [-v] [-d dialect] [-dot dir] file...")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() == 0 {
		fs.Usage()
		return 2
	}
	if err := draw(fs.Args(), *dialect, *dir, *verbose, stderr); err != nil {
		fmt.Fprintln(stderr, "genseq:", err)
		return 1
	}
	return 0
}

func draw(paths []string, dialect, dir string, verbose bool, stderr io.Writer) error {
	tabs, err := compile(paths, dialect)
	if err != nil {
		return err
	}
	for _, t := range tabs {
		if verbose {
			fmt.Fprintf(stderr, "genseq: %s, %d states, waits for %s\n", t.Name, len(t.Next)-1, strings.Join(t.Waiting(), " "))
		}
		if dir == "" {
			continue
		}
		b, err := seqdraw.Draw(t)
		if err == nil {
			err = os.MkdirAll(dir, 0o755)
		}
		if err == nil {
			err = os.WriteFile(filepath.Join(dir, t.Name+".dot"), b, 0o644)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func compile(paths []string, dialect string) ([]*seq.Table, error) {
	type parsed struct {
		tree *seq.Grammar
		tab  *seq.Table
	}
	var all []parsed
	var trees []*seq.Grammar
	for _, path := range paths {
		d := dialect
		if d == "" {
			d = "hotkey"
			if filepath.Ext(path) == ".tree" {
				d = "keypad"
			}
		}
		f, err := os.Open(path)
		if err != nil {
			return nil, err
		}
		switch d {
		case "keypad":
			g, err := keypad.Parse(f, path)
			f.Close()
			if err != nil {
				return nil, err
			}
			trees = append(trees, g)
			all = append(all, parsed{tree: g})
		case "hotkey":
			r, err := hotkey.Parse(f, path, nil)
			f.Close()
			if err != nil {
				return nil, err
			}
			all = append(all, parsed{tab: r.Table})
		default:
			f.Close()
			return nil, fmt.Errorf("no dialect called %s", d)
		}
	}

	a := keypad.Alphabet(trees...)
	var compiled []*seq.Table
	var out []*seq.Table
	for _, p := range all {
		if p.tab != nil {
			out = append(out, p.tab)
			continue
		}
		t, err := seq.Compile(p.tree, a)
		if err != nil {
			return nil, err
		}
		if len(t.Conflicts) > 0 {
			return nil, errors.New(strings.Join(t.Conflicts, "\n"))
		}
		compiled = append(compiled, t)
		out = append(out, t)
	}
	if len(compiled) > 0 {
		if _, err := seq.Link(compiled, "trees", keypad.Answered); err != nil {
			return nil, err
		}
	}
	return out, nil
}
