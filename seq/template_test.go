package seq_test

import (
	"testing"

	"github.com/cptaffe/automaton/seq"
	"github.com/cptaffe/automaton/seq/seqtest"
)

func calls(t *testing.T) (*seq.Table, *seq.Alphabet) {
	t.Helper()
	a := seq.NewAlphabet([]string{"a", "b", "c", "d"}, "answered")
	return compile(t, &seq.Grammar{
		Name:   "g",
		Scopes: []string{"start", "other", "dialing"},
		Params: map[string]string{"dialing": "then"},
		Rules: []seq.Rule{
			goes(rule(a, 1, "start", "a", "call"), "dialing", "other"),
			goes(rule(a, 2, "start", "b", "back"), "dialing", "this"),
			rule(a, 3, "other", "c", "other"),
			goes(rule(a, 4, "dialing", "answered", ""), "then", ""),
		},
		Any: []seq.Any{
			{Scope: "dialing", Pattern: "b", Elem: elems(a, "b")[0], Action: "gave-up", Goto: "then", Line: 5},
			{Pattern: "d", Elem: elems(a, "d")[0], Action: "hush", Line: 6},
		},
	}, a), a
}

func TestATemplateIsCompiledOncePerArgument(t *testing.T) {
	tab, _ := calls(t)

	names := scopeNames(tab)

	if names != "start dialing(other) dialing(start) other" {
		t.Errorf("the scopes are %q; want start dialing(other) dialing(start) other", names)
	}
}

func TestAParameterIsWhereTheScopeGoesNext(t *testing.T) {
	tab, a := calls(t)
	for _, c := range []struct {
		name  string
		given []seqtest.Step
		then  string
	}{
		{
			name:  "into the template",
			given: []seqtest.Step{{Key: "a", Taken: true, Fired: "call"}, {Key: "c"}},
			then:  "dialing(other)",
		},
		{
			name:  "out by a rule to the argument",
			given: []seqtest.Step{{Key: "a", Taken: true, Fired: "call"}, {Key: "answered", Taken: true}, {Key: "c", Taken: true, Fired: "other"}},
			then:  "other",
		},
		{
			name:  "out by a wildcard to the argument",
			given: []seqtest.Step{{Key: "a", Taken: true, Fired: "call"}, {Key: "b", Taken: true, Fired: "gave-up"}},
			then:  "other",
		},
		{
			name:  "back to where this was",
			given: []seqtest.Step{{Key: "b", Taken: true, Fired: "back"}, {Key: "answered", Taken: true}},
			then:  "start",
		},
		{
			name:  "a wildcard of every scope in the template",
			given: []seqtest.Step{{Key: "a", Taken: true, Fired: "call"}, {Key: "d", Taken: true, Fired: "hush"}},
			then:  "dialing(other)",
		},
		{
			name:  "a wildcard of every scope outside the template",
			given: []seqtest.Step{{Key: "d", Taken: true, Fired: "hush"}},
			then:  "start",
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			m := seqtest.Machine(t, tab, "")

			seqtest.Run(t, m, a, nil, c.given)

			if m.Scope() != c.then {
				t.Errorf("the machine is in %s; want %s", m.Scope(), c.then)
			}
		})
	}
}

func TestATemplateMayGoBackToItselfByItsParameter(t *testing.T) {
	a := seq.NewAlphabet([]string{"a", "b"})
	g := &seq.Grammar{Name: "g", Scopes: []string{"start", "t"}, Params: map[string]string{"t": "then"},
		Rules: []seq.Rule{
			goes(rule(a, 1, "start", "a", "x"), "t", "start"),
			goes(rule(a, 2, "t", "a", "x"), "t", "then"),
			goes(rule(a, 3, "t", "b", "x"), "then", ""),
		}}

	tab := compile(t, g, a)

	if names := scopeNames(tab); names != "start t(start)" {
		t.Errorf("the scopes are %q; want start t(start)", names)
	}
}

func TestAWildcardOfEveryScopeHoldsWhereNoTemplateIsUsed(t *testing.T) {
	a := seq.NewAlphabet([]string{"a", "b"})
	g := &seq.Grammar{Name: "g", Scopes: []string{"start", "t"}, Params: map[string]string{"t": "then"},
		Rules: []seq.Rule{goes(rule(a, 1, "t", "a", "x"), "then", "")},
		Any:   []seq.Any{{Pattern: "b", Elem: elems(a, "b")[0], Action: "hush", Line: 2}}}
	tab := compile(t, g, a)
	m := seqtest.Machine(t, tab, "")

	seqtest.Run(t, m, a, nil, []seqtest.Step{{Key: "b", Taken: true, Fired: "hush"}})
}
