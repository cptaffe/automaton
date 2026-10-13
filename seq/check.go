package seq

import "fmt"

type move struct {
	from string
	to   string
	arg  string
	line int
}

func (r Rule) move() move { return move{r.Scope, r.Goto, r.Arg, r.Line} }
func (x Any) move() move  { return move{x.Scope, x.Goto, x.Arg, x.Line} }

type checker struct {
	g        *Grammar
	declared map[string]bool
}

func Check(g *Grammar) error {
	c := &checker{g: g, declared: map[string]bool{}}
	if err := c.scopes(); err != nil {
		return err
	}
	for _, r := range g.Rules {
		if err := c.rule(r); err != nil {
			return err
		}
	}
	for _, x := range g.Any {
		if err := c.move(x.move()); err != nil {
			return err
		}
	}
	return cycles(g)
}

func (c *checker) scopes() error {
	g := c.g
	if len(g.Scopes) == 0 {
		return fmt.Errorf("%s: no scopes, so there is nowhere to begin", g.Name)
	}
	for _, s := range g.Scopes {
		if s == "this" {
			return fmt.Errorf("%s: a scope cannot be called this, which names the scope a rule is in", g.Name)
		}
		if c.declared[s] {
			return fmt.Errorf("%s: scope %s is declared twice", g.Name, s)
		}
		c.declared[s] = true
	}
	if p := g.Params[g.Scopes[0]]; p != "" {
		return fmt.Errorf("%s: %s takes a parameter, and the first scope is where a machine begins", g.Name, g.Scopes[0])
	}
	return nil
}

func (c *checker) rule(r Rule) error {
	if !c.declared[r.Scope] {
		return fmt.Errorf("%s:%d: the rule is in %s, and there is no such scope", c.g.Name, r.Line, r.Scope)
	}
	if err := c.move(r.move()); err != nil {
		return err
	}
	if err := c.body(r); err != nil {
		return err
	}
	if r.Ahead < 0 || r.Ahead >= len(r.Elems) && r.Include == "" {
		return fmt.Errorf("%s:%d: %q reads ahead of every key it has, and would never consume one", c.g.Name, r.Line, r.Pattern)
	}
	return nil
}

func (c *checker) body(r Rule) error {
	g := c.g
	switch {
	case r.Include == "" && len(r.Elems) == 0:
		return fmt.Errorf("%s:%d: a rule with no pattern", g.Name, r.Line)
	case r.Include == "":
		return nil
	case !c.declared[r.Include]:
		return fmt.Errorf("%s:%d: -> %s, and there is no such scope", g.Name, r.Line, r.Include)
	case g.Params[r.Include] != "":
		return fmt.Errorf("%s:%d: -> %s reads %s's rules here, and %s takes a parameter", g.Name, r.Line, r.Include, r.Include, r.Include)
	case len(r.Elems) > 0 || r.Action != "" || r.Goto != "":
		return fmt.Errorf("%s:%d: -> %s with no pattern reads %s's rules, and does nothing else", g.Name, r.Line, r.Include, r.Include)
	}
	return nil
}

func (c *checker) move(m move) error {
	g := c.g
	param := g.Params[m.from]
	switch {
	case m.to == "":
		return nil
	case m.to == param && m.arg != "":
		return fmt.Errorf("%s:%d: %s is %s's parameter and cannot take one of its own", g.Name, m.line, m.to, m.from)
	case m.to == param:
		return nil
	case !c.declared[m.to]:
		return fmt.Errorf("%s:%d: -> %s, and there is no such scope", g.Name, m.line, m.to)
	case g.Params[m.to] != "" && m.arg == "":
		return fmt.Errorf("%s:%d: %s takes a parameter and was given none", g.Name, m.line, m.to)
	case g.Params[m.to] == "" && m.arg != "":
		return fmt.Errorf("%s:%d: %s takes no parameter and was given %s", g.Name, m.line, m.to, m.arg)
	}
	return c.arg(m, param)
}

func (c *checker) arg(m move, param string) error {
	g := c.g
	switch {
	case m.arg == "", m.arg == "this", m.arg == param:
		return nil
	case !c.declared[m.arg]:
		return fmt.Errorf("%s:%d: -> %s(%s), and there is no scope %s", g.Name, m.line, m.to, m.arg, m.arg)
	case g.Params[m.arg] != "":
		return fmt.Errorf("%s:%d: -> %s(%s), and %s takes a parameter of its own", g.Name, m.line, m.to, m.arg, m.arg)
	}
	return nil
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
