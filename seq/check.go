package seq

import "fmt"

func Check(g *Grammar) error {
	if len(g.Scopes) == 0 {
		return fmt.Errorf("%s: no scopes, so there is nowhere to begin", g.Name)
	}
	seen := map[string]bool{}
	for _, s := range g.Scopes {
		seen[s] = true
	}
	for _, r := range g.Rules {
		if r.Goto != "" && !seen[r.Goto] {
			return fmt.Errorf("%s:%d: -> %s, and there is no such scope", g.Name, r.Line, r.Goto)
		}
	}
	for _, a := range g.Any {
		if a.Goto != "" && !seen[a.Goto] {
			return fmt.Errorf("%s:%d: -> %s, and there is no such scope", g.Name, a.Line, a.Goto)
		}
	}
	for _, r := range g.Rules {
		n := len(r.Elems)
		if n == 0 || !r.Elems[n-1].Many {
			continue
		}
		if _, ok := g.Settles[r.Scope]; !ok {
			return fmt.Errorf("%s:%d: %q ends in a repeat, and %s has no %%settles to say where it stops", g.Name, r.Line, r.Pattern, r.Scope)
		}
	}
	return nil
}
