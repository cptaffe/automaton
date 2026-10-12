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
			ok, fired = m.Step(sym(tab.Alpha, s.key), Env{"app": s.app})
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
	m.Step(sym(a, "a"), nil)
	if got := strings.Join(m.Waits(), " "); got != "hangup settle" {
		t.Errorf("after a, the machine waits for %q", got)
	}
	if got := actions(m.Settle(nil)); got != "one" {
		t.Errorf("settling after a fired %q; want one", got)
	}
	m.Step(sym(a, "a"), nil)
	if _, f := m.Step(sym(a, "hangup"), nil); actions(f) != "hung" {
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

func TestAnUnknownKeyIsTheCallers(t *testing.T) {
	tab := modes(t)
	m := machine(t, tab, "")
	env := Env{"app": "devdraw"}
	m.Step(sym(tab.Alpha, "x"), env)
	if _, ok := tab.Alpha.Of("f13"); ok {
		t.Fatal("f13 is in the alphabet")
	}
	if f := m.Settle(env); actions(f) != "show" || m.Pending() {
		t.Errorf("settling fired %q and left pending %v; want show and nothing pending", actions(f), m.Pending())
	}
}

func TestOnlyAPlaceAMachineCanBeIsCompiled(t *testing.T) {
	tab := modes(t)
	if scopeNames(tab) != "main marks" {
		t.Errorf("the scopes are %v; acme and safari are only read in", tab.Scopes)
	}
	if _, err := NewMachine(tab, "acme"); err == nil {
		t.Error("a machine began in a scope nothing goes to")
	}
}

func hk(t *testing.T, a *Alphabet, g *Grammar) *Table {
	t.Helper()
	tab, err := Compile(g, a)
	if err != nil {
		t.Fatal(err)
	}
	return tab
}

func TestOverlappingRepeatsCompile(t *testing.T) {
	a := NewAlphabet([]string{"1", "2"}, "T")
	one := keys(a, "1")
	g := &Grammar{Name: "g", Scopes: []string{"s"}, Delays: map[string]time.Duration{"T": time.Second}, Settles: map[string]string{"s": "T"},
		Rules: []Rule{{Scope: "s", Pattern: "1+1+2", Elems: []Class{many(one), many(one), keys(a, "2")}, Action: "x"}}}
	done := make(chan error, 1)
	go func() { _, err := Compile(g, a); done <- err }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("1+1+2 is still compiling")
	}
}

func TestACompleteRuleAnotherOriginExtendsSettles(t *testing.T) {
	a := NewAlphabet([]string{"x", "s"}, "settle")
	g := &Grammar{Name: "g", Scopes: []string{"main", "a", "b"},
		Delays:  map[string]time.Duration{"settle": time.Second},
		Settles: map[string]string{"a": "settle"},
		Rules: []Rule{
			{Scope: "main", Include: "a"},
			{Scope: "main", Include: "b"},
			{Scope: "a", Pattern: "x", Elems: []Class{keys(a, "x")}, Action: "one"},
			{Scope: "b", Pattern: "x s", Elems: []Class{keys(a, "x"), keys(a, "s")}, Action: "two"},
		}}
	tab := hk(t, a, g)
	m := machine(t, tab, "")
	m.Step(sym(a, "x"), nil)
	if got := strings.Join(m.Waits(), " "); got != "settle" {
		t.Errorf("after x the machine waits for %q; want settle", got)
	}
	if _, f := m.Step(sym(a, "settle"), nil); actions(f) != "one" {
		t.Errorf("settle fired %q; want one", actions(f))
	}

	delete(g.Settles, "a")
	tab = hk(t, a, g)
	m = machine(t, tab, "")
	if _, f := m.Step(sym(a, "x"), nil); actions(f) != "one" || m.Pending() {
		t.Errorf("with nothing to settle on, x fired %q and pending is %v; want one at once", actions(f), m.Pending())
	}
}

func TestOneScopeReadInTwiceIsNoConflict(t *testing.T) {
	a := NewAlphabet([]string{"s"})
	g := &Grammar{Name: "g", Scopes: []string{"main", "emacs"}, Rules: []Rule{
		{Scope: "main", Include: "emacs", When: app("Emacs")},
		{Scope: "main", Include: "emacs", When: app("Terminal")},
		{Scope: "emacs", Pattern: "s", Elems: []Class{keys(a, "s")}, Action: "save"},
	}}
	tab := hk(t, a, g)
	if len(tab.Conflicts) > 0 {
		t.Fatal(tab.Conflicts)
	}
	for _, name := range []string{"Emacs", "Terminal"} {
		m := machine(t, tab, "")
		if ok, f := m.Step(sym(a, "s"), Env{"app": name}); !ok || actions(f) != "save" {
			t.Errorf("in %s, s took %v and fired %q", name, ok, actions(f))
		}
	}
}

func TestAWildcardSettlesWhatWasPending(t *testing.T) {
	a := NewAlphabet([]string{"x", "s", "j"}, "settle")
	j := keys(a, "j")
	tab := hk(t, a, &Grammar{Name: "g", Scopes: []string{"m"},
		Delays:  map[string]time.Duration{"settle": time.Second},
		Settles: map[string]string{"m": "settle"},
		Rules: []Rule{
			{Scope: "m", Pattern: "x", Elems: []Class{keys(a, "x")}, Action: "one"},
			{Scope: "m", Pattern: "x s", Elems: []Class{keys(a, "x"), keys(a, "s")}, Action: "two"},
		},
		Any: []Any{{Scope: "m", Pattern: "j", Elem: j, Action: "wild"}}})
	m := machine(t, tab, "")
	m.Step(sym(a, "x"), nil)
	if _, f := m.Step(sym(a, "j"), nil); actions(f) != "one,wild" {
		t.Errorf("j after x fired %q; want one,wild", actions(f))
	}
}

func TestARuleOnTheSettleEventConflicts(t *testing.T) {
	a := NewAlphabet([]string{"1"}, "T")
	tab := hk(t, a, &Grammar{Name: "g", Scopes: []string{"s"},
		Delays: map[string]time.Duration{"T": time.Second}, Settles: map[string]string{"s": "T"},
		Rules: []Rule{
			{Scope: "s", Pattern: "1", Elems: []Class{keys(a, "1")}, Action: "one"},
			{Scope: "s", Pattern: "1 T", Elems: []Class{keys(a, "1"), event("T")}, Action: "two"},
		}})
	if len(tab.Conflicts) == 0 || !strings.Contains(strings.Join(tab.Conflicts, "\n"), "T settles here and a rule reads T next") {
		t.Errorf("conflicts %v", tab.Conflicts)
	}
}

func scopeNames(tab *Table) string {
	var names []string
	for _, s := range tab.Scopes {
		names = append(names, s.Name)
	}
	return strings.Join(names, " ")
}
