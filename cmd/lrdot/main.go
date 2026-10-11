package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/cptaffe/automaton/dot"
	"github.com/cptaffe/automaton/lr"
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
	shape := make([]dot.Rule, len(g.Rules))

	for i, r := range g.Rules {
		shape[i] = dot.Rule{Name: r.Name, Alts: r.Alts, Start: i == 0}
	}

	if err := os.WriteFile(os.Args[2], dot.Shape(shape, name), 0o644); err != nil {
		fail(err)
	}

	over := ""

	if t := g.Tokens(name); t != nil && !shapeOnly {
		t.SetNote(note(name, t, dot.Nested(shape)))
		at := strings.TrimSuffix(os.Args[2], ".dot") + "-tokens.dot"

		if err := os.WriteFile(at, dot.Draw(t), 0o644); err != nil {
			fail(err)
		}

		over = fmt.Sprintf(", %d states over tokens", t.States())
	}

	fmt.Printf("lrdot: %s, %d rules%s\n", name, len(g.Rules), over)
}

func note(name string, t *lr.Tokens, nested []string) string {
	head := fmt.Sprintf("%s over tokens -- %s, %s", name, plural(t.States(), "state"), plural(t.Symbols(), "token"))

	if len(nested) == 0 {
		return head + "; nothing self-embeds, so this is exactly the grammar"
	}

	sort.Strings(nested)
	verb := "self-embed"

	if len(nested) == 1 {
		verb = "self-embeds"
	}

	return fmt.Sprintf("%s; %s %s, so this reads more than the grammar does", head, strings.Join(nested, ", "), verb)
}

func plural(n int, what string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, what)
	}

	return fmt.Sprintf("%d %ss", n, what)
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "lrdot:", err)
	os.Exit(1)
}
