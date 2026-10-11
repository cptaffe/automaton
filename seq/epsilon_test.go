package seq

import (
	"strings"
	"testing"
	"time"
)

func app(name string) Cond {
	c, err := NewCompare("app", Eq, name)
	if err != nil {
		panic(err)
	}
	return c
}

func modes(t *testing.T) *Table {
	t.Helper()
	a := NewAlphabet([]string{"s", "x", "m", "esc", "k"}, "settle")
	g := &Grammar{
		Name:    "keys",
		Scopes:  []string{"main", "acme", "safari", "marks"},
		Delays:  map[string]time.Duration{"settle": time.Second},
		Settles: map[string]string{"acme": "settle"},
		Rules: []Rule{
			{Scope: "main", Include: "acme", When: app("devdraw"), Line: 1},
			{Scope: "main", Include: "safari", When: app("Safari"), Line: 2},
			{Scope: "acme", Pattern: "s", Elems: []Class{keys(a, "s")}, Action: "put", Line: 3},
			{Scope: "acme", Pattern: "x", Elems: []Class{keys(a, "x")}, Action: "show", Line: 4},
			{Scope: "acme", Pattern: "x s", Elems: []Class{keys(a, "x"), keys(a, "s")}, Action: "save", Line: 5},
			{Scope: "acme", Pattern: "m", Elems: []Class{keys(a, "m")}, Goto: "marks", Line: 6},
			{Scope: "safari", Pattern: "s", Elems: []Class{keys(a, "s")}, Action: "share", Line: 7},
			{Scope: "marks", Pattern: "k", Elems: []Class{keys(a, "k")}, Action: "mark", Line: 8},
			{Scope: "marks", Pattern: "esc", Elems: []Class{keys(a, "esc")}, Goto: "main", Line: 9},
		},
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

type step struct {
	key, app string
	ok       bool
	fired    string
}

func run(t *testing.T, tab *Table, steps []step) {
	t.Helper()
	m := machine(t, tab, "")
	for i, s := range steps {
		var ok bool
		var fired []*Accept
		if s.key == "" {
			fired = m.Settle(Env{"app": s.app})
		} else {
			ok, fired = m.Step(tab.Alpha.Of(s.key), Env{"app": s.app})
		}
		if ok != s.ok || actions(fired) != s.fired {
			t.Errorf("step %d, %s in %s: took %v and fired %q; want %v and %q", i, s.key, s.app, ok, actions(fired), s.ok, s.fired)
		}
	}
}

func TestAGuardedEpsilonChoosesByApp(t *testing.T) {
	run(t, modes(t), []step{
		{"s", "devdraw", true, "put"},
		{"s", "Safari", true, "share"},
		{"s", "Finder", false, ""},
		{"x", "Safari", false, ""},
		{"k", "devdraw", false, ""},
	})
}

func TestAPrefixIsNotTakenWhereItsRuleDoesNotHold(t *testing.T) {
	run(t, modes(t), []step{
		{"x", "Finder", false, ""},
		{"x", "devdraw", true, ""},
		{"s", "devdraw", true, "save"},
		{"x", "devdraw", true, ""},
		{"settle", "devdraw", true, "show"},
	})
}

func TestAModeIsLeftByGoingBack(t *testing.T) {
	tab := modes(t)
	run(t, tab, []step{
		{"m", "devdraw", true, ""},
		{"k", "Safari", true, "mark"},
		{"s", "Safari", false, ""},
		{"esc", "devdraw", true, ""},
		{"s", "Safari", true, "share"},
		{"k", "devdraw", false, ""},
	})
}

func TestSettlingFiresTheCompleteRule(t *testing.T) {
	a := NewAlphabet([]string{"a", "b"}, "hangup", "settle")
	g := &Grammar{Name: "g", Scopes: []string{"s"},
		Delays:  map[string]time.Duration{"hangup": time.Second, "settle": time.Second},
		Settles: map[string]string{"s": "settle"},
		Rules: []Rule{
			{Scope: "s", Pattern: "a", Elems: []Class{keys(a, "a")}, Action: "one"},
			{Scope: "s", Pattern: "a b", Elems: []Class{keys(a, "a"), keys(a, "b")}, Action: "two"},
			{Scope: "s", Pattern: "a <hangup>", Elems: []Class{keys(a, "a"), event("hangup")}, Action: "hung"},
		}}
	tab, err := Compile(g, a)
	if err != nil {
		t.Fatal(err)
	}
	m := machine(t, tab, "")
	m.Step(a.Of("a"), nil)
	if got := strings.Join(m.Waits(), " "); got != "hangup settle" {
		t.Errorf("after a, the machine waits for %q", got)
	}
	if got := actions(m.Settle(nil)); got != "one" {
		t.Errorf("settling after a fired %q; want one", got)
	}
	m.Step(a.Of("a"), nil)
	if _, f := m.Step(a.Of("hangup"), nil); actions(f) != "hung" {
		t.Errorf("a then hangup fired %q; want hung", actions(f))
	}
}

func TestACompleteRuleAnEventCanFollowIsAmbiguous(t *testing.T) {
	a := NewAlphabet([]string{"a"}, "hangup")
	g := &Grammar{Name: "g", Scopes: []string{"s"},
		Delays: map[string]time.Duration{"hangup": time.Second},
		Rules: []Rule{
			{Scope: "s", Pattern: "a", Elems: []Class{keys(a, "a")}, Action: "one", Line: 1},
			{Scope: "s", Pattern: "a <hangup>", Elems: []Class{keys(a, "a"), event("hangup")}, Action: "hung", Line: 2},
		}}
	tab, err := Compile(g, a)
	if err != nil {
		t.Fatal(err)
	}
	if len(tab.Conflicts) != 1 || !strings.Contains(tab.Conflicts[0], `g:1: "a" is complete, and a <hangup> can still follow`) {
		t.Errorf("the conflicts are %v", tab.Conflicts)
	}
}

func TestAnUnknownScopeIsAnError(t *testing.T) {
	tab := compile(t)
	if _, err := NewMachine(tab, "nosuch"); err == nil {
		t.Error("a machine began in a scope that is not there")
	}
	a := alphabet()
	if _, err := Compile(&Grammar{Name: "g", Scopes: []string{"s"}, Rules: []Rule{{Scope: "s", Pattern: "a", Elems: []Class{keys(a, "a")}, Goto: "typo"}}}, a); err == nil {
		t.Error("a -> to no scope compiled")
	}
}

func TestAnEpsilonIsChecked(t *testing.T) {
	a := alphabet()
	for _, c := range []struct {
		rules []Rule
		want  string
	}{
		{[]Rule{{Scope: "s", Include: "nowhere"}}, "no such scope"},
		{[]Rule{{Scope: "s", Include: "t"}, {Scope: "t", Include: "s"}}, "goes round in a loop"},
		{[]Rule{{Scope: "s", Include: "t", Action: "x"}}, "does nothing else"},
		{[]Rule{{Scope: "s", Action: "x"}}, "no pattern"},
	} {
		g := &Grammar{Name: "g", Scopes: []string{"s", "t"}, Rules: append(c.rules, Rule{Scope: "t", Pattern: "a", Elems: []Class{keys(a, "a")}, Action: "a"})}
		if err := Check(g); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("got %v; want %q", err, c.want)
		}
	}
}

func TestAnUnknownKeySettlesAndPasses(t *testing.T) {
	tab := modes(t)
	m := machine(t, tab, "")
	env := Env{"app": "devdraw"}
	m.Step(tab.Alpha.Of("x"), env)
	if ok, f := m.Step(0, env); ok || actions(f) != "show" {
		t.Errorf("a key no rule names took %v and fired %q; want it passed and show settled", ok, actions(f))
	}
	if m.Pending() {
		t.Error("the machine still waits after an unknown key")
	}
}
