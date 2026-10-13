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
		c.Syms = append(c.Syms, sym(a, n))
	}
	c.Text = strings.Join(names, "|")
	return c
}

func sym(a *Alphabet, name string) Sym {
	s, ok := a.Of(name)
	if !ok {
		panic("no symbol " + name)
	}
	return s
}

func many(c Class) Class { c.Many = true; return c }

func event(name string) Class { return Class{Event: name, Text: name} }

func grammar(a *Alphabet) *Grammar {
	return &Grammar{
		Name:   "g",
		Scopes: []string{"start", "mode"},
		Delays: map[string]time.Duration{"T": time.Second},
		Rules: []Rule{
			{Scope: "start", Pattern: "a T", Elems: []Class{keys(a, "a"), event("T")}, Action: "one", Line: 1},
			{Scope: "start", Pattern: "a b", Elems: []Class{keys(a, "a"), keys(a, "b")}, Action: "two", Line: 2},
			{Scope: "start", Pattern: "c d+ T", Elems: []Class{keys(a, "c"), many(keys(a, "d", "e")), event("T")}, Action: "run", Line: 3},
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
	tab, err := Compile(grammar(a), a)
	if err != nil {
		t.Fatal(err)
	}
	if len(tab.Conflicts) > 0 {
		t.Fatal(tab.Conflicts)
	}
	return tab
}

func actions(as []*Accept) string {
	s := make([]string, 0, len(as))
	for _, a := range as {
		s = append(s, a.Action)
	}
	return strings.Join(s, ",")
}

func machine(t *testing.T, tab *Table, scope string) *Machine {
	t.Helper()
	m, err := NewMachine(tab, scope)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func TestAlphabet(t *testing.T) {
	a := NewAlphabet([]string{"x", "y", "x"}, "late", "early")
	if a.Size() != 4 || a.Keys() != 2 {
		t.Errorf("size %d keys %d", a.Size(), a.Keys())
	}
	if s, _ := a.Of("late"); s != 2 || a.Name(3) != "early" || a.IsKey(2) || !a.IsEvent(2) {
		t.Errorf("events are not kept in order after the keys: %v", a.names)
	}
	if _, ok := a.Of("nope"); ok || a.Name(4) != "" || a.Name(-1) != "" || a.IsKey(4) || a.IsEvent(4) {
		t.Error("a symbol past the alphabet has a name")
	}
}

func TestMachine(t *testing.T) {
	tab := compile(t)
	m := machine(t, tab, "")
	on := Env{"on": "1"}
	for _, c := range []struct {
		in    string
		ok    bool
		fired string
		waits []string
	}{
		{"a", true, "", []string{"T"}},
		{"b", true, "two", nil},
		{"a", true, "", []string{"T"}},
		{"T", true, "one", nil},
		{"c", true, "", nil},
		{"d", true, "", []string{"T"}},
		{"e", true, "", []string{"T"}},
		{"T", true, "run", nil},
		{"b", false, "", nil},
		{"m", true, "", nil},
		{"x", true, "ex", nil},
		{"z", true, "zap", nil},
		{"x", false, "", nil},
	} {
		ok, fired := m.Step(sym(tab.Alpha, c.in), on)
		if ok != c.ok || actions(fired) != c.fired || !reflect.DeepEqual(m.Waits(), c.waits) {
			t.Errorf("%s: got %v %q %v, want %v %q %v", c.in, ok, actions(fired), m.Waits(), c.ok, c.fired, c.waits)
		}
	}
	if m.Scope() != "start" {
		t.Errorf("scope %q after the recovery's goto", m.Scope())
	}
}

func TestAFailedWhenLeavesTheKey(t *testing.T) {
	tab := compile(t)
	m := machine(t, tab, "mode")
	if ok, fired := m.Step(sym(tab.Alpha, "x"), Env{}); ok || len(fired) != 0 {
		t.Errorf("a rule whose when fails consumed %v and fired %v", ok, actions(fired))
	}
}

func TestOperand(t *testing.T) {
	tab := compile(t)
	m := machine(t, tab, "")
	var last []*Accept
	for _, n := range []string{"c", "d", "d", "T"} {
		_, last = m.Step(sym(tab.Alpha, n), nil)
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
	if !strings.Contains(joined, "two rules, one key sequence") || !strings.Contains(joined, "can still follow") {
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
	} {
		if err := Check(c.g); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("got %v, want %q", err, c.want)
		}
	}
	if _, err := Compile(&Grammar{Name: "g", Scopes: []string{"s"}, Rules: []Rule{{Scope: "s", Elems: []Class{event("nope")}}}}, a); err == nil {
		t.Error("an undeclared event compiled")
	}
	tab, err := Compile(&Grammar{Name: "g", Scopes: []string{"s"}, Rules: []Rule{{Scope: "s", Pattern: "a+", Action: "x", Elems: []Class{many(keys(a, "a"))}}}}, a)
	if err != nil || len(tab.Conflicts) != 1 || !strings.Contains(tab.Conflicts[0], `"a+" is complete, and a+ can still follow`) {
		t.Errorf("a pattern that ends in a repeat compiled to %v, %v", tab, err)
	}
}

func TestCond(t *testing.T) {
	re, err := NewCompare("app", Match, "^(a|b)$")
	if err != nil {
		t.Fatal(err)
	}
	eq, err := NewCompare("app", Eq, "c")
	if err != nil {
		t.Fatal(err)
	}
	c := AnyOf(AllOf(Name("x"), Not{Of: re}), eq)
	for app, want := range map[string]bool{"a": false, "c": true, "d": true} {
		if got := c.Eval(Env{"x": "1", "app": app}); got != want {
			t.Errorf("app=%s: %v", app, got)
		}
	}
	if got := c.Names(); !reflect.DeepEqual(got, []string{"app", "x"}) {
		t.Errorf("names %v", got)
	}
	if _, err := NewCompare("a", NoOp, "b"); err == nil {
		t.Error("no op compared")
	}
	if (&Compare{}).Eval(Env{}) {
		t.Error("the zero comparison holds")
	}
	if s := c.String(); !strings.Contains(s, "not app ~") {
		t.Errorf("string %q", s)
	}
}
