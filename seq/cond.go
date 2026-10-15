package seq

import (
	"fmt"
	"regexp"
	"slices"
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

type Op int

const (
	NoOp Op = iota
	Eq
	Ne
	Match
)

func (o Op) String() string {
	switch o {
	case Eq:
		return "=="
	case Ne:
		return "!="
	case Match:
		return "~"
	}
	return "?"
}

type Compare struct {
	name  string
	op    Op
	value string
	re    *regexp.Regexp
}

func NewCompare(name string, op Op, value string) (*Compare, error) {
	c := &Compare{name: name, op: op, value: value}
	switch op {
	case Eq, Ne:
	case Match:
		re, err := regexp.Compile(value)
		if err != nil {
			return nil, err
		}
		c.re = re
	default:
		return nil, fmt.Errorf("%v is not a comparison", op)
	}
	return c, nil
}

func (c *Compare) Eval(e Env) bool {
	v := e[c.name]
	switch c.op {
	case Eq:
		return v == c.value
	case Ne:
		return v != c.value
	case Match:
		return c.re.MatchString(v)
	}
	return false
}

func (c *Compare) Names() []string { return []string{c.name} }
func (c *Compare) String() string  { return fmt.Sprintf("%s %v %q", c.name, c.op, c.value) }

func AllOf(a, b Cond) Cond {
	switch x := a.(type) {
	case nil:
		return b
	case And:
		if b == nil {
			return a
		}
		return append(x[:len(x):len(x)], b)
	}
	if b == nil {
		return a
	}
	return And{a, b}
}

func AnyOf(a, b Cond) Cond {
	if x, ok := a.(Or); ok {
		return append(x[:len(x):len(x)], b)
	}
	return Or{a, b}
}

func Holds(c Cond, e Env) bool { return c == nil || c.Eval(e) }

func names(cs []Cond) []string {
	var out []string
	for _, c := range cs {
		out = append(out, c.Names()...)
	}
	slices.Sort(out)
	return slices.Compact(out)
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
