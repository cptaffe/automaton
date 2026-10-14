package seq_test

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/cptaffe/automaton/internal/seqtest"
	"github.com/cptaffe/automaton/seq"
)

func app(name string) seq.Cond {
	c, err := seq.NewCompare("app", seq.Eq, name)
	if err != nil {
		panic(err)
	}
	return c
}

func modes(t *testing.T) *seq.Table {
	t.Helper()
	a := seq.NewAlphabet([]string{"s", "x", "m", "esc", "k"}, "settle")
	g := &seq.Grammar{
		Name:   "keys",
		Scopes: []string{"main", "acme", "safari", "marks"},
		Delays: map[string]time.Duration{"settle": time.Second},
		Rules: []seq.Rule{
			{Scope: "main", Include: "acme", When: app("devdraw"), Line: 1},
			{Scope: "main", Include: "safari", When: app("Safari"), Line: 2},
			rule(a, 3, "acme", "s", "put"),
			rule(a, 4, "acme", "x settle", "show"),
			rule(a, 5, "acme", "x s", "save"),
			goes(rule(a, 6, "acme", "m", ""), "marks", ""),
			rule(a, 7, "safari", "s", "share"),
			rule(a, 8, "marks", "k", "mark"),
			goes(rule(a, 9, "marks", "esc", ""), "main", ""),
		},
	}
	return compile(t, g, a)
}

var (
	devdraw = seq.Env{"app": "devdraw"}
	safari  = seq.Env{"app": "Safari"}
	finder  = seq.Env{"app": "Finder"}
)

func TestAGuardedEpsilonChoosesByApp(t *testing.T) {
	tab := modes(t)
	m := seqtest.Machine(t, tab, "")

	seqtest.Run(t, m, tab.Alpha, nil, []seqtest.Step{
		{Key: "s", Env: devdraw, Taken: true, Fired: "put"},
		{Key: "s", Env: safari, Taken: true, Fired: "share"},
		{Key: "s", Env: finder},
		{Key: "x", Env: safari},
		{Key: "k", Env: devdraw},
	})
}

func TestAPrefixIsNotTakenWhereItsRuleDoesNotHold(t *testing.T) {
	tab := modes(t)
	m := seqtest.Machine(t, tab, "")

	seqtest.Run(t, m, tab.Alpha, nil, []seqtest.Step{
		{Key: "x", Env: finder},
		{Key: "x", Env: devdraw, Taken: true, Pending: true},
		{Key: "s", Env: devdraw, Taken: true, Fired: "save"},
		{Key: "x", Env: devdraw, Taken: true, Pending: true},
		{Key: "settle", Env: devdraw, Taken: true, Fired: "show"},
	})
}

func TestAModeIsLeftByGoingBack(t *testing.T) {
	tab := modes(t)
	m := seqtest.Machine(t, tab, "")

	seqtest.Run(t, m, tab.Alpha, nil, []seqtest.Step{
		{Key: "m", Env: devdraw, Taken: true},
		{Key: "k", Env: safari, Taken: true, Fired: "mark"},
		{Key: "s", Env: safari},
		{Key: "esc", Env: devdraw, Taken: true},
		{Key: "s", Env: safari, Taken: true, Fired: "share"},
		{Key: "k", Env: devdraw},
	})
}

func TestAMachineBeginsOnlyInACompiledScope(t *testing.T) {
	tab := modes(t)
	for _, c := range []struct {
		name  string
		given string
		then  string
		err   string
	}{
		{name: "the first scope by default", given: "", then: "main"},
		{name: "a scope a rule goes to", given: "marks", then: "marks"},
		{name: "a scope only read in", given: "acme", err: `keys has no scope "acme"`},
		{name: "a scope that is not there", given: "nosuch", err: `keys has no scope "nosuch"`},
	} {
		t.Run(c.name, func(t *testing.T) {
			m, err := seq.NewMachine(tab, c.given)

			switch {
			case c.err != "" && (err == nil || err.Error() != c.err):
				t.Errorf("NewMachine(%q) = %v; want %s", c.given, err, c.err)
			case c.err == "" && err != nil:
				t.Errorf("NewMachine(%q) = %v", c.given, err)
			case c.err == "" && m.Scope() != c.then:
				t.Errorf("NewMachine(%q) begins in %s; want %s", c.given, m.Scope(), c.then)
			}
		})
	}
}

func TestOnlyAPlaceAMachineCanBeIsCompiled(t *testing.T) {
	tab := modes(t)

	names := scopeNames(tab)

	if names != "main marks" {
		t.Errorf("the scopes are %q; want main marks, as acme and safari are only read in", names)
	}
}

func TestADelayEndsARule(t *testing.T) {
	a := seq.NewAlphabet([]string{"a", "b"}, "hangup", "settle")
	tab := compile(t, &seq.Grammar{Name: "g", Scopes: []string{"s"},
		Delays: map[string]time.Duration{"hangup": time.Second, "settle": time.Second},
		Rules: []seq.Rule{
			rule(a, 1, "s", "a settle", "one"),
			rule(a, 2, "s", "a b", "two"),
			rule(a, 3, "s", "a hangup", "hung"),
		}}, a)
	m := seqtest.Machine(t, tab, "")

	seqtest.Run(t, m, a, nil, seqtest.Prefix("a"))
	waits := m.Waits()
	seqtest.Run(t, m, a, nil, []seqtest.Step{
		{Key: "settle", Taken: true, Fired: "one"},
		{Key: "a", Taken: true, Pending: true},
		{Key: "hangup", Taken: true, Fired: "hung"},
		{Key: "hangup"},
	})

	if want := []string{"hangup", "settle"}; !reflect.DeepEqual(waits, want) {
		t.Errorf("after a, the machine waits for %q; want %q", waits, want)
	}
}

func TestResetDropsAPrefix(t *testing.T) {
	tab := modes(t)
	m := seqtest.Machine(t, tab, "")
	seqtest.Run(t, m, tab.Alpha, devdraw, seqtest.Prefix("x"))

	m.Reset()

	seqtest.Run(t, m, tab.Alpha, devdraw, []seqtest.Step{{Key: "settle"}})
}

func TestOverlappingRepeatsMatch(t *testing.T) {
	a := seq.NewAlphabet([]string{"1", "2"})
	tab := compile(t, &seq.Grammar{Name: "g", Scopes: []string{"s"}, Rules: []seq.Rule{rule(a, 1, "s", "1+ 1+ 2", "x")}}, a)
	for _, c := range []struct {
		name string
		then []seqtest.Step
	}{
		{name: "one of each", then: append(seqtest.Prefix("1 1"), seqtest.Step{Key: "2", Taken: true, Fired: "x"})},
		{name: "more of the first", then: append(seqtest.Prefix("1 1 1 1"), seqtest.Step{Key: "2", Taken: true, Fired: "x"})},
		{name: "too few", then: append(seqtest.Prefix("1"), seqtest.Step{Key: "2"})},
	} {
		t.Run(c.name, func(t *testing.T) {
			seqtest.Run(t, seqtest.Machine(t, tab, ""), a, nil, c.then)
		})
	}
}

func TestACompleteRuleFiresThoughAnotherScopeGoesOn(t *testing.T) {
	a := seq.NewAlphabet([]string{"x", "s"})
	tab := compile(t, &seq.Grammar{Name: "g", Scopes: []string{"main", "a", "b"},
		Rules: []seq.Rule{
			{Scope: "main", Include: "a"},
			{Scope: "main", Include: "b"},
			rule(a, 1, "a", "x", "one"),
			rule(a, 2, "b", "x s", "two"),
		}}, a)
	m := seqtest.Machine(t, tab, "")

	seqtest.Run(t, m, a, nil, []seqtest.Step{
		{Key: "x", Taken: true, Fired: "one"},
		{Key: "s"},
	})
}

func TestOneScopeReadInTwiceIsNoConflict(t *testing.T) {
	a := seq.NewAlphabet([]string{"s"})
	tab := compile(t, &seq.Grammar{Name: "g", Scopes: []string{"main", "emacs"}, Rules: []seq.Rule{
		{Scope: "main", Include: "emacs", When: app("Emacs")},
		{Scope: "main", Include: "emacs", When: app("Terminal")},
		rule(a, 1, "emacs", "s", "save"),
	}}, a)
	for _, c := range []struct {
		name  string
		given seq.Env
		then  seqtest.Step
	}{
		{name: "the first app", given: seq.Env{"app": "Emacs"}, then: seqtest.Step{Key: "s", Taken: true, Fired: "save"}},
		{name: "the second app", given: seq.Env{"app": "Terminal"}, then: seqtest.Step{Key: "s", Taken: true, Fired: "save"}},
		{name: "neither app", given: seq.Env{"app": "Finder"}, then: seqtest.Step{Key: "s"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			seqtest.Run(t, seqtest.Machine(t, tab, ""), a, c.given, []seqtest.Step{c.then})
		})
	}
}

func TestAWildcardDropsAPrefix(t *testing.T) {
	a := seq.NewAlphabet([]string{"x", "s", "j"}, "settle")
	tab := compile(t, &seq.Grammar{Name: "g", Scopes: []string{"m"},
		Delays: map[string]time.Duration{"settle": time.Second},
		Rules: []seq.Rule{
			rule(a, 1, "m", "x settle", "one"),
			rule(a, 2, "m", "x s", "two"),
		},
		Any: []seq.Any{{Scope: "m", Pattern: "j", Elem: elems(a, "j")[0], Action: "wild"}}}, a)
	m := seqtest.Machine(t, tab, "")

	seqtest.Run(t, m, a, nil, []seqtest.Step{
		{Key: "x", Taken: true, Pending: true},
		{Key: "j", Taken: true, Fired: "wild"},
		{Key: "s"},
	})
}

func scopeNames(tab *seq.Table) string {
	names := make([]string, 0, len(tab.Scopes))
	for _, s := range tab.Scopes {
		names = append(names, s.Name)
	}
	return strings.Join(names, " ")
}
