package seq

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

type Env map[string]string

type Cond interface {
	Eval(Env) bool
	Names() []string
	String() string
}

type Name string

func (n Name) Eval(e Env) bool { return e[string(n)] != "" }
func (n Name) Names() []string { return []string{string(n)} }
func (n Name) String() string  { return string(n) }

type Not struct{ Of Cond }

func (n Not) Eval(e Env) bool { return !n.Of.Eval(e) }
func (n Not) Names() []string { return n.Of.Names() }
func (n Not) String() string  { return "not " + wrap(n.Of) }

type And []Cond

func (a And) Eval(e Env) bool {
	for _, c := range a {
		if !c.Eval(e) {
			return false
		}
	}
	return true
}

func (a And) Names() []string { return names(a) }
func (a And) String() string  { return join(a, " and ") }

type Or []Cond

func (o Or) Eval(e Env) bool {
	for _, c := range o {
		if c.Eval(e) {
			return true
		}
	}
	return false
}

func (o Or) Names() []string { return names(o) }
func (o Or) String() string  { return join(o, " or ") }

type Compare struct {
	Name  string
	Op    string
	Value string

	re *regexp.Regexp
}

func NewCompare(name, op, value string) (*Compare, error) {
	c := &Compare{Name: name, Op: op, Value: value}
	switch op {
	case "==", "!=":
	case "~":
		re, err := regexp.Compile(value)
		if err != nil {
			return nil, err
		}
		c.re = re
	default:
		return nil, fmt.Errorf("%s is not a comparison", op)
	}
	return c, nil
}

func (c *Compare) Eval(e Env) bool {
	v := e[c.Name]
	switch c.Op {
	case "==":
		return v == c.Value
	case "!=":
		return v != c.Value
	}
	return c.re.MatchString(v)
}

func (c *Compare) Names() []string { return []string{c.Name} }
func (c *Compare) String() string  { return fmt.Sprintf("%s %s %q", c.Name, c.Op, c.Value) }

func names(cs []Cond) []string {
	seen := map[string]bool{}
	var out []string
	for _, c := range cs {
		for _, n := range c.Names() {
			if !seen[n] {
				seen[n] = true
				out = append(out, n)
			}
		}
	}
	sort.Strings(out)
	return out
}

func join(cs []Cond, sep string) string {
	s := make([]string, len(cs))
	for i, c := range cs {
		s[i] = wrap(c)
	}
	return strings.Join(s, sep)
}

func wrap(c Cond) string {
	switch c.(type) {
	case And, Or:
		return "(" + c.String() + ")"
	}
	return c.String()
}
