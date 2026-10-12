package seq

import (
	"strings"
	"testing"
)

func calls(t *testing.T, scopes []string, params map[string]string, rules ...Rule) (*Table, error) {
	t.Helper()
	a := NewAlphabet([]string{"a", "b", "c"}, "answered")
	for i := range rules {
		if rules[i].Include == "" && len(rules[i].Elems) == 0 {
			p := rules[i].Pattern
			if a.IsEvent(a.Of(p)) {
				rules[i].Elems = []Class{event(p)}
			} else {
				rules[i].Elems = []Class{keys(a, p)}
			}
		}
	}
	return Compile(&Grammar{Name: "g", Scopes: scopes, Params: params, Rules: rules}, a)
}

func TestAParameterIsWhereTheScopeGoesNext(t *testing.T) {
	tab, err := calls(t, []string{"start", "other", "dialing"}, map[string]string{"dialing": "then"},
		Rule{Scope: "start", Pattern: "a", Action: "call", Goto: "dialing", Arg: "other"},
		Rule{Scope: "start", Pattern: "b", Action: "back", Goto: "dialing", Arg: "this"},
		Rule{Scope: "other", Pattern: "c", Action: "other"},
		Rule{Scope: "dialing", Pattern: "answered", Goto: "then"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if got := scopeNames(tab); got != "start dialing(other) dialing(start) other" {
		t.Errorf("the scopes are %q", got)
	}
	m := machine(t, tab, "")
	for _, c := range []struct{ sym, scope string }{
		{"a", "dialing(other)"}, {"c", "dialing(other)"}, {"answered", "other"},
	} {
		m.Step(tab.Alpha.Of(c.sym), nil)
		if m.Scope() != c.scope {
			t.Errorf("after %s the machine is in %s; want %s", c.sym, m.Scope(), c.scope)
		}
	}
	m = machine(t, tab, "")
	m.Step(tab.Alpha.Of("b"), nil)
	m.Step(tab.Alpha.Of("answered"), nil)
	if m.Scope() != "start" {
		t.Errorf("-> dialing(this) came back to %s", m.Scope())
	}
}

func TestATemplateCannotHoldItself(t *testing.T) {
	_, err := calls(t, []string{"start", "t"}, map[string]string{"t": "then"},
		Rule{Scope: "start", Pattern: "a", Goto: "t", Arg: "start"},
		Rule{Scope: "t", Pattern: "a", Goto: "t", Arg: "this"},
		Rule{Scope: "t", Pattern: "b", Goto: "then"},
	)
	if err == nil || !strings.Contains(err.Error(), "t(t(start)) holds t inside itself") {
		t.Errorf("t calling t(this) compiled: %v", err)
	}
	if _, err := calls(t, []string{"start", "t"}, map[string]string{"t": "then"},
		Rule{Scope: "start", Pattern: "a", Goto: "t", Arg: "start"},
		Rule{Scope: "t", Pattern: "a", Goto: "t", Arg: "then"},
		Rule{Scope: "t", Pattern: "b", Goto: "then"},
	); err != nil {
		t.Errorf("t going back to itself with its own parameter: %v", err)
	}
	if _, err := calls(t, []string{"this"}, nil, Rule{Scope: "this", Pattern: "a", Action: "x"}); err == nil {
		t.Error("a scope called this compiled")
	}
}
