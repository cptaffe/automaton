package seq

import "fmt"

func Check(g *Grammar) error {
	if len(g.Scopes) == 0 {
		return fmt.Errorf("%s: no scopes, so there is nowhere to begin", g.Name)
	}
	seen := map[string]bool{}
	for _, s := range g.Scopes {
		if s == "this" {
			return fmt.Errorf("%s: a scope cannot be called this, which names the scope a rule is in", g.Name)
		}
		if seen[s] {
			return fmt.Errorf("%s: scope %s is declared twice", g.Name, s)
		}
		seen[s] = true
	}
	if p := g.Params[g.Scopes[0]]; p != "" {
		return fmt.Errorf("%s: %s takes a parameter, and the first scope is where a machine begins", g.Name, g.Scopes[0])
	}
	move := func(scope, to, arg string, line int) error {
		param := g.Params[scope]
		switch {
		case to == "":
			return nil
		case to == param && arg != "":
			return fmt.Errorf("%s:%d: %s is %s's parameter and cannot take one of its own", g.Name, line, to, scope)
		case to == param:
			return nil
		case !seen[to]:
			return fmt.Errorf("%s:%d: -> %s, and there is no such scope", g.Name, line, to)
		case g.Params[to] != "" && arg == "":
			return fmt.Errorf("%s:%d: %s takes a parameter and was given none", g.Name, line, to)
		case g.Params[to] == "" && arg != "":
			return fmt.Errorf("%s:%d: %s takes no parameter and was given %s", g.Name, line, to, arg)
		case arg == "this":
			return nil
		case arg != "" && arg != param && !seen[arg]:
			return fmt.Errorf("%s:%d: -> %s(%s), and there is no scope %s", g.Name, line, to, arg, arg)
		case arg != "" && arg != param && g.Params[arg] != "":
			return fmt.Errorf("%s:%d: -> %s(%s), and %s takes a parameter of its own", g.Name, line, to, arg, arg)
		}
		return nil
	}
	for _, r := range g.Rules {
		if !seen[r.Scope] {
			return fmt.Errorf("%s:%d: the rule is in %s, and there is no such scope", g.Name, r.Line, r.Scope)
		}
		if err := move(r.Scope, r.Goto, r.Arg, r.Line); err != nil {
			return err
		}
		switch {
		case r.Include != "" && !seen[r.Include]:
			return fmt.Errorf("%s:%d: -> %s, and there is no such scope", g.Name, r.Line, r.Include)
		case r.Include != "" && g.Params[r.Include] != "":
			return fmt.Errorf("%s:%d: -> %s reads %s's rules here, and %s takes a parameter", g.Name, r.Line, r.Include, r.Include, r.Include)
		case r.Include != "" && (len(r.Elems) > 0 || r.Action != "" || r.Goto != ""):
			return fmt.Errorf("%s:%d: -> %s with no pattern reads %s's rules, and does nothing else", g.Name, r.Line, r.Include, r.Include)
		case r.Include == "" && len(r.Elems) == 0:
			return fmt.Errorf("%s:%d: a rule with no pattern", g.Name, r.Line)
		case r.Ahead < 0 || r.Ahead >= len(r.Elems) && r.Include == "":
			return fmt.Errorf("%s:%d: %q reads ahead of every key it has, and would never consume one", g.Name, r.Line, r.Pattern)
		}
	}
	for _, a := range g.Any {
		if err := move(a.Scope, a.Goto, a.Arg, a.Line); err != nil {
			return err
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
