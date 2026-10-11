package seq

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

func keys(a *Alphabet, names ...string) Class {
	var c Class
	for _, n := range names {
		s, _ := a.Of(n)
		c.Syms = append(c.Syms, s)
	}
	c.Text = strings.Join(names, "|")
	return c
}

func many(c Class) Class { c.Many = true; return c }

func event(name string) Class { return Class{Event: name, Text: name} }

func grammar(a *Alphabet) *Grammar {
	return &Grammar{
		Name:    "g",
		Scopes:  []string{"start", "mode"},
		Delays:  map[string]time.Duration{"T": time.Second},
		Settles: map[string]string{"start": "T"},
		Rules: []Rule{
			{Scope: "start", Pattern: "a", Elems: []Class{keys(a, "a")}, Action: "one", Line: 1},
			{Scope: "start", Pattern: "a b", Elems: []Class{keys(a, "a"), keys(a, "b")}, Action: "two", Line: 2},
			{Scope: "start", Pattern: "c d+", Elems: []Class{keys(a, "c"), many(keys(a, "d", "e"))}, Action: "run", Line: 3},
			{Scope: "start", Pattern: "m", Elems: []Class{keys(a, "m")}, Goto: "mode", Line: 4},
			{Scope: "mode", Pattern: "x", Elems: []Class{keys(a, "x")}, Action: "ex", When: Name("on"), Line: 5},
		},
		Any: []Any{{Scope: "", Pattern: "z", Elem: keys(a, "z"), Action: "zap", Goto: "start", Line: 6}},
	}
}

func alphabet() *Alphabet {
	return NewAlphabet([]string{"a", "b", "c", "d", "e", "m", "x", "z"}, "T")
}

func compile(t *testing.T) *Table {
	t.Helper()
	a := alphabet()
	g := grammar(a)
	if err := Check(g); err != nil {
		t.Fatal(err)
	}
	tab, err := Compile(g, a)
	if err != nil {
		t.Fatal(err)
	}
	if len(tab.Conflicts) > 0 {
		t.Fatal(tab.Conflicts)
	}
	return tab
}

func actions(as []*Accept) string {
	var s []string
	for _, a := range as {
		s = append(s, a.Action)
	}
	return strings.Join(s, ",")
}

func TestAlphabet(t *testing.T) {
	a := NewAlphabet([]string{"x", "y", "x"}, "late", "early")
	if a.Size() != 4 || a.Keys() != 2 {
		t.Errorf("size %d keys %d", a.Size(), a.Keys())
	}
	if s, _ := a.Of("late"); s != 2 || a.Name(3) != "early" || a.IsKey(2) {
		t.Errorf("events are not kept in order after the keys: %v", a.names)
	}
}

func TestMachine(t *testing.T) {
	tab := compile(t)
	sym := func(n string) Sym { s, _ := tab.Alpha.Of(n); return s }
	m := NewMachine(tab, "start")
	on := Env{"on": "1"}
	for _, c := range []struct {
		in     string
		ok     bool
		fired  string
		waits  []string
		settle bool
	}{
		{"a", true, "", []string{"T"}, false},
		{"b", true, "two", nil, false},
		{"a", true, "", []string{"T"}, false},
		{"", false, "one", nil, true},
		{"c", true, "", nil, false},
		{"d", true, "", []string{"T"}, false},
		{"e", true, "", []string{"T"}, false},
		{"T", true, "run", nil, false},
		{"b", false, "", nil, false},
		{"m", true, "", nil, false},
		{"x", true, "ex", nil, false},
		{"z", true, "zap", nil, false},
		{"x", false, "", nil, false},
	} {
		if c.settle {
			if got := actions(m.Settle(on)); got != c.fired {
				t.Errorf("settle fired %q, want %q", got, c.fired)
			}
			continue
		}
		ok, fired := m.Step(sym(c.in), on)
		if ok != c.ok || actions(fired) != c.fired || !reflect.DeepEqual(m.Waits(), c.waits) {
			t.Errorf("%s: got %v %q %v, want %v %q %v", c.in, ok, actions(fired), m.Waits(), c.ok, c.fired, c.waits)
		}
	}
	if m.Scope() != "start" {
		t.Errorf("scope %q after the recovery's goto", m.Scope())
	}
}

func TestWhenRefuses(t *testing.T) {
	tab := compile(t)
	m := NewMachine(tab, "mode")
	x, _ := tab.Alpha.Of("x")
	if ok, fired := m.Step(x, Env{}); !ok || len(fired) != 0 {
		t.Errorf("a rule whose when fails consumed %v and fired %v", ok, actions(fired))
	}
}

func TestOperand(t *testing.T) {
	tab := compile(t)
	m := NewMachine(tab, "start")
	var last []*Accept
	for _, n := range []string{"c", "d", "d", "T"} {
		s, _ := tab.Alpha.Of(n)
		_, last = m.Step(s, nil)
	}
	if len(last) != 1 || last[0].Operand == nil || *last[0].Operand != (Slice{From: 1, To: 0}) {
		t.Errorf("operand %+v", last)
	}
}

func TestConflicts(t *testing.T) {
	a := alphabet()
	g := &Grammar{Name: "g", Scopes: []string{"s"}, Rules: []Rule{
		{Scope: "s", Pattern: "a", Elems: []Class{keys(a, "a")}, Action: "x"},
		{Scope: "s", Pattern: "a", Elems: []Class{keys(a, "a")}, Action: "y"},
		{Scope: "s", Pattern: "b", Elems: []Class{keys(a, "b")}, Action: "x"},
		{Scope: "s", Pattern: "b c", Elems: []Class{keys(a, "b"), keys(a, "c")}, Action: "y"},
	}}
	tab, err := Compile(g, a)
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(tab.Conflicts, "\n")
	if !strings.Contains(joined, "two rules, one key sequence") || !strings.Contains(joined, "no %settles") {
		t.Errorf("conflicts %v", tab.Conflicts)
	}
}

func TestCheck(t *testing.T) {
	a := alphabet()
	for _, c := range []struct {
		g    *Grammar
		want string
	}{
		{&Grammar{Name: "g"}, "no scopes"},
		{&Grammar{Name: "g", Scopes: []string{"s"}, Rules: []Rule{{Scope: "s", Goto: "t", Elems: []Class{keys(a, "a")}}}}, "no such scope"},
		{&Grammar{Name: "g", Scopes: []string{"s"}, Rules: []Rule{{Scope: "s", Action: "x", Elems: []Class{many(keys(a, "a"))}}}}, "ends in a repeat"},
	} {
		if err := Check(c.g); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("got %v, want %q", err, c.want)
		}
	}
	if _, err := Compile(&Grammar{Name: "g", Scopes: []string{"s"}, Rules: []Rule{{Scope: "s", Elems: []Class{event("nope")}}}}, a); err == nil {
		t.Error("an undeclared event compiled")
	}
}

func TestLink(t *testing.T) {
	a := NewAlphabet([]string{"1", "2"}, "answered")
	one, _ := a.Of("1")
	two, _ := a.Of("2")
	first := &Grammar{Name: "first", Scopes: []string{"s"}, Rules: []Rule{{Scope: "s", Pattern: "1", Elems: []Class{{Syms: []Sym{one}}}, Action: "dial", Hands: "second"}}}
	second := &Grammar{Name: "second", Scopes: []string{"s"}, Rules: []Rule{{Scope: "s", Pattern: "2", Elems: []Class{{Syms: []Sym{two}}}, Action: "menu"}}}
	t1, _ := Compile(first, a)
	t2, _ := Compile(second, a)
	tab, err := Link([]*Table{t2, t1}, "all", "answered")
	if err != nil {
		t.Fatal(err)
	}
	m := NewMachine(tab, "first.s")
	ans, _ := a.Of("answered")
	if _, f := m.Step(one, nil); actions(f) != "dial" {
		t.Fatalf("first tree fired %q", actions(f))
	}
	at := tab.Start["first.s"]
	n := tab.Next[tab.Next[at][one]][ans]
	if n == Dead || tab.Scope[n] != "second.s" {
		t.Errorf("answered after dial leads to %d", n)
	}
	if _, err := Link([]*Table{t1}, "all", "answered"); err == nil {
		t.Error("a hand-off to a missing tree linked")
	}
}

func TestCond(t *testing.T) {
	re, err := NewCompare("app", "~", "^(a|b)$")
	if err != nil {
		t.Fatal(err)
	}
	c := Or{And{Name("x"), Not{Of: re}}, &Compare{Name: "app", Op: "==", Value: "c"}}
	for app, want := range map[string]bool{"a": false, "c": true, "d": true} {
		if got := c.Eval(Env{"x": "1", "app": app}); got != want {
			t.Errorf("app=%s: %v", app, got)
		}
	}
	if got := c.Names(); !reflect.DeepEqual(got, []string{"app", "x"}) {
		t.Errorf("names %v", got)
	}
	if _, err := NewCompare("a", "<", "b"); err == nil {
		t.Error("< compared")
	}
	if s := c.String(); !strings.Contains(s, "not app ~") {
		t.Errorf("string %q", s)
	}
}
