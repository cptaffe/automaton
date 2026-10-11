package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cptaffe/automaton/dot"
	"github.com/cptaffe/automaton/lr"
	"github.com/cptaffe/automaton/lr/draw"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "usage: lrdot <y.output> <out.dot> [shape-only]")
		os.Exit(1)
	}
	shapeOnly := len(os.Args) > 3 && os.Args[3] == "shape-only"

	g, err := lr.Read(os.Args[1])
	if err != nil {
		fail(err)
	}
	name := strings.TrimPrefix(strings.TrimSuffix(filepath.Base(os.Args[2]), ".dot"), "parse-")
	if err := os.WriteFile(os.Args[2], draw.Shape(g, name), 0o644); err != nil {
		fail(err)
	}

	over := ""
	if t := draw.Tokens(g, name); t != nil && !shapeOnly {
		at := strings.TrimSuffix(os.Args[2], ".dot") + "-tokens.dot"
		if err := os.WriteFile(at, dot.Draw(t), 0o644); err != nil {
			fail(err)
		}
		over = fmt.Sprintf(", %d states over tokens", t.States())
	}
	fmt.Printf("lrdot: %s, %d rules%s\n", name, len(g.Rules), over)
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "lrdot:", err)
	os.Exit(1)
}
