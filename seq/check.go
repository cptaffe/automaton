package seq

import "fmt"

func Check(g *Grammar) error {
	if len(g.Scopes) == 0 {
		return fmt.Errorf("%s: no scopes, so there is nowhere to begin", g.Name)
	}
	seen := map[string]bool{}
	for _, s := range g.Scopes {
		if seen[s] {
			return fmt.Errorf("%s: scope %s is declared twice", g.Name, s)
		}
		seen[s] = true
	}
	for _, r := range g.Rules {
		switch {
		case !seen[r.Scope]:
			return fmt.Errorf("%s:%d: the rule is in %s, and there is no such scope", g.Name, r.Line, r.Scope)
		case r.Goto != "" && !seen[r.Goto]:
			return fmt.Errorf("%s:%d: -> %s, and there is no such scope", g.Name, r.Line, r.Goto)
		case r.Include != "" && !seen[r.Include]:
			return fmt.Errorf("%s:%d: -> %s, and there is no such scope", g.Name, r.Line, r.Include)
		case r.Include != "" && (len(r.Elems) > 0 || r.Action != "" || r.Goto != "" || r.Hands != ""):
			return fmt.Errorf("%s:%d: -> %s with no pattern reads %s's rules, and does nothing else", g.Name, r.Line, r.Include, r.Include)
		case r.Include == "" && len(r.Elems) == 0:
			return fmt.Errorf("%s:%d: a rule with no pattern", g.Name, r.Line)
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
	return cycles(g)
}

func cycles(g *Grammar) error {
	state := map[string]int{}
	var visit func(at string) error
	visit = func(at string) error {
		state[at] = 1
		for _, r := range g.Rules {
			if r.Scope != at || r.Include == "" {
				continue
			}
			switch state[r.Include] {
			case 1:
				return fmt.Errorf("%s:%d: -> %s goes round in a loop back to %s", g.Name, r.Line, r.Include, at)
			case 0:
				if err := visit(r.Include); err != nil {
					return err
				}
			}
		}
		state[at] = 2
		return nil
	}
	for _, s := range g.Scopes {
		if state[s] == 0 {
			if err := visit(s); err != nil {
				return err
			}
		}
	}
	return nil
}
