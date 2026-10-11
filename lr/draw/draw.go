package draw

import (
	"fmt"
	"sort"
	"strings"

	"github.com/cptaffe/automaton/dot"
	"github.com/cptaffe/automaton/lr"
)

func rules(g *lr.Grammar) []dot.Rule {
	out := make([]dot.Rule, len(g.Rules))
	for i, r := range g.Rules {
		out[i] = dot.Rule{Name: r.Name, Alts: r.Alts, Start: i == 0}
	}
	return out
}

func Shape(g *lr.Grammar, name string) []byte {
	return dot.Shape(rules(g), name)
}

func Tokens(g *lr.Grammar, name string) *lr.Tokens {
	t := g.Tokens(name)
	if t == nil {
		return nil
	}
	head := fmt.Sprintf("%s over tokens -- %s, %s", name, dot.Plural(t.States(), "state"), dot.Plural(t.Symbols(), "token"))
	nested := dot.Nested(rules(g))
	if len(nested) == 0 {
		t.SetNote(head + "; nothing self-embeds, so this is exactly the grammar")
		return t
	}
	sort.Strings(nested)
	verb := "self-embed"
	if len(nested) == 1 {
		verb = "self-embeds"
	}
	t.SetNote(fmt.Sprintf("%s; %s %s, so this reads more than the grammar does", head, strings.Join(nested, ", "), verb))
	return t
}
