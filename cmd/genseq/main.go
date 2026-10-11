package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cptaffe/automaton/seq"
	"github.com/cptaffe/automaton/seq/draw"
	"github.com/cptaffe/automaton/seq/hotkey"
	"github.com/cptaffe/automaton/seq/keypad"
)

func main() {
	dialect := flag.String("d", "", "the dialect, keypad or hotkey; by default .tree is keypad and anything else hotkey")
	dir := flag.String("dot", "", "write Graphviz into this directory")
	flag.Parse()

	tabs, err := compile(flag.Args(), *dialect)
	if err != nil {
		fmt.Fprintln(os.Stderr, "genseq:", err)
		os.Exit(1)
	}
	for _, t := range tabs {
		fmt.Printf("genseq: %s, %d rules in %d states\n", t.Name, len(t.Rules), len(t.Next))
		for _, name := range t.Waiting() {
			fmt.Printf("genseq:   waits %s in state %d\n", name, t.WaitsIn(name))
		}
		if *dir == "" {
			continue
		}
		at := filepath.Join(*dir, t.Name+".dot")
		if err := os.MkdirAll(*dir, 0o755); err != nil {
			fmt.Fprintln(os.Stderr, "genseq:", err)
			os.Exit(1)
		}
		if err := os.WriteFile(at, draw.Draw(t), 0o644); err != nil {
			fmt.Fprintln(os.Stderr, "genseq:", err)
			os.Exit(1)
		}
		fmt.Printf("genseq:   %s\n", at)
	}
}

func compile(paths []string, dialect string) ([]*seq.Table, error) {
	var tabs []*seq.Table
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
		case "hotkey":
			r, err := hotkey.Parse(f, path, nil)
			f.Close()
			if err != nil {
				return nil, err
			}
			tabs = append(tabs, r.Table)
		default:
			f.Close()
			return nil, fmt.Errorf("no dialect called %s", d)
		}
	}
	a := keypad.Alphabet(trees...)
	for _, g := range trees {
		t, err := seq.Compile(g, a)
		if err != nil {
			return nil, err
		}
		if len(t.Conflicts) > 0 {
			return nil, fmt.Errorf("%s: %s", g.Name, strings.Join(t.Conflicts, "; "))
		}
		tabs = append(tabs, t)
	}
	return tabs, nil
}
