package main

import (
	"github.com/cptaffe/automaton/dot"
	"github.com/cptaffe/automaton/lexgen"
)

func shapeOf(f *lexgen.Flat) []byte {
	var rs []dot.Rule

	for i, s := range f.Scopes {
		r := dot.Rule{Name: s.Name, Start: i == 0}

		if s.Includes != "" {
			r.Alts = append(r.Alts, "includes "+s.Includes)
		}

		for _, rule := range f.Rules {
			if rule.Scope == s.Name {
				r.Alts = append(r.Alts, rule.Text)
			}
		}

		rs = append(rs, r)
	}

	return dot.Shape(rs, f.Name)
}
