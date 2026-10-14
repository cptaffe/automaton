package seq_test

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/cptaffe/automaton/fa"
	"github.com/cptaffe/automaton/internal/seqtest"
	"github.com/cptaffe/automaton/seq"
)

func elems(a *seq.Alphabet, pattern string) []seq.Class {
	fields := strings.Fields(pattern)
	out := make([]seq.Class, 0, len(fields))
	for _, f := range fields {
		text := f
		many := strings.HasSuffix(f, "+")
		f = strings.TrimSuffix(f, "+")
		var c seq.Class
		switch s, _ := a.Of(f); {
		case f == "any":
			c = seq.Class{Any: true}
		case a.IsEvent(s):
			c = seq.Class{Event: f}
		default:
			for _, name := range strings.Split(f, "|") {
				s, ok := a.Of(name)
				if !ok {
					panic("no symbol " + name)
				}
				c.Syms = append(c.Syms, s)
			}
		}
		c.Many, c.Text = many, text
		out = append(out, c)
	}
	return out
}

func rule(a *seq.Alphabet, line int, scope, pattern, action string) seq.Rule {
	return seq.Rule{Scope: scope, Pattern: pattern, Elems: elems(a, pattern), Action: action, Line: line}
}

func alphabet() *seq.Alphabet {
	return seq.NewAlphabet([]string{"a", "b", "c", "d", "e", "m", "x", "z"}, "T", "U")
}

func grammar(a *seq.Alphabet) *seq.Grammar {
	mode := rule(a, 4, "start", "m", "")
	mode.Goto = "mode"
	ex := rule(a, 5, "mode", "x", "ex")
	ex.When = seq.Name("on")
	return &seq.Grammar{
		Name:   "g",
		Scopes: []string{"start", "mode"},
		Delays: map[string]time.Duration{"T": time.Second, "U": time.Second},
		Rules: []seq.Rule{
			rule(a, 1, "start", "a T", "one"),
			rule(a, 2, "start", "a b", "two"),
			rule(a, 3, "start", "c d|e+ T", "run"),
			mode,
			ex,
		},
		Any: []seq.Any{{Pattern: "z", Elem: elems(a, "z")[0], Action: "zap", Goto: "start", Line: 6}},
	}
}

func compile(t *testing.T, g *seq.Grammar, a *seq.Alphabet) *seq.Table {
	t.Helper()
	tab, err := seq.Compile(g, a)
	if err != nil {
		t.Fatal(err)
	}
	if len(tab.Conflicts) > 0 {
		t.Fatal(tab.Conflicts)
	}
	return tab
}

type symbol struct {
	name       string
	key, event bool
}

func TestAnAlphabetNumbersKeysThenEvents(t *testing.T) {
	a := seq.NewAlphabet([]string{"x", "y", "x"}, "late", "early", "late")
	for _, c := range []struct {
		name  string
		given seq.Sym
		then  symbol
	}{
		{name: "before the first", given: -1, then: symbol{}},
		{name: "the first key", given: 0, then: symbol{name: "x", key: true}},
		{name: "the last key", given: 1, then: symbol{name: "y", key: true}},
		{name: "the first event", given: 2, then: symbol{name: "late", event: true}},
		{name: "the last event", given: 3, then: symbol{name: "early", event: true}},
		{name: "past the last", given: 4, then: symbol{}},
	} {
		t.Run(c.name, func(t *testing.T) {
			got := symbol{name: a.Name(c.given), key: a.IsKey(c.given), event: a.IsEvent(c.given)}

			if got != c.then {
				t.Errorf("symbol %d is %+v; want %+v", c.given, got, c.then)
			}
		})
	}
}

func TestAnAlphabetNamesEachSymbolOnce(t *testing.T) {
	a := seq.NewAlphabet([]string{"x", "y", "x"}, "late", "early", "late")

	size, keys := a.Size(), a.Keys()

	if size != 4 || keys != 2 {
		t.Errorf("the alphabet has %d symbols and %d keys; want 4 and 2", size, keys)
	}
}

func TestAnAlphabetFindsASymbolByName(t *testing.T) {
	a := seq.NewAlphabet([]string{"x", "y", "x"}, "late", "early", "late")
	for _, c := range []struct {
		name  string
		given string
		then  seq.Sym
		found bool
	}{
		{name: "a key", given: "y", then: 1, found: true},
		{name: "a key named twice", given: "x", then: 0, found: true},
		{name: "an event", given: "early", then: 3, found: true},
		{name: "an event named twice", given: "late", then: 2, found: true},
		{name: "no symbol", given: "nope", then: 0, found: false},
	} {
		t.Run(c.name, func(t *testing.T) {
			s, ok := a.Of(c.given)

			if s != c.then || ok != c.found {
				t.Errorf("Of(%q) = %d, %v; want %d, %v", c.given, s, ok, c.then, c.found)
			}
		})
	}
}

func TestAMachineWalksItsRules(t *testing.T) {
	a := alphabet()
	tab := compile(t, grammar(a), a)
	m := seqtest.Machine(t, tab, "")

	seqtest.Run(t, m, a, seq.Env{"on": "1"}, []seqtest.Step{
		{Key: "a", Taken: true, Pending: true},
		{Key: "b", Taken: true, Fired: "two"},
		{Key: "a", Taken: true, Pending: true},
		{Key: "T", Taken: true, Fired: "one"},
		{Key: "c", Taken: true, Pending: true},
		{Key: "d", Taken: true, Pending: true},
		{Key: "e", Taken: true, Pending: true},
		{Key: "T", Taken: true, Fired: "run"},
		{Key: "b"},
		{Key: "T"},
		{Key: "m", Taken: true},
		{Key: "x", Taken: true, Fired: "ex"},
		{Key: "a"},
		{Key: "z", Taken: true, Fired: "zap"},
		{Key: "x"},
	})

	if m.Scope() != "start" {
		t.Errorf("the machine is in %s after z; want start", m.Scope())
	}
}

func TestAPendingMachineWaitsForTheDelaysItCanRead(t *testing.T) {
	a := alphabet()
	tab := compile(t, grammar(a), a)
	for _, c := range []struct {
		name  string
		given string
		then  []string
	}{
		{name: "nothing pressed", given: "", then: nil},
		{name: "a key that a delay ends", given: "a", then: []string{"T"}},
		{name: "a key that more keys follow", given: "c", then: nil},
		{name: "a repeat that a delay ends", given: "c d e", then: []string{"T"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			m := seqtest.Machine(t, tab, "")
			seqtest.Run(t, m, a, nil, seqtest.Prefix(c.given))

			waits := m.Waits()

			if !reflect.DeepEqual(waits, c.then) {
				t.Errorf("after %q the machine waits for %q; want %q", c.given, waits, c.then)
			}
		})
	}
}

func TestATableWaitsForTheDelaysItReads(t *testing.T) {
	a := alphabet()
	tab := compile(t, grammar(a), a)

	waiting := tab.Waiting()

	if want := []string{"T"}; !reflect.DeepEqual(waiting, want) {
		t.Errorf("the table waits for %q; want %q", waiting, want)
	}
}

func TestAFailedWhenLeavesTheKey(t *testing.T) {
	a := alphabet()
	tab := compile(t, grammar(a), a)
	m := seqtest.Machine(t, tab, "mode")

	seqtest.Run(t, m, a, seq.Env{}, []seqtest.Step{{Key: "x"}})
}

func TestAPeekFiresThoughNothingReadsTheKeyAgain(t *testing.T) {
	a := alphabet()
	show := rule(a, 1, "s", "x any", "show")
	show.Ahead = 1
	tab := compile(t, &seq.Grammar{Name: "g", Scopes: []string{"s"}, Rules: []seq.Rule{show, rule(a, 2, "s", "a", "put")}}, a)
	m := seqtest.Machine(t, tab, "")

	seqtest.Run(t, m, a, nil, []seqtest.Step{
		{Key: "x", Taken: true, Pending: true},
		{Key: "b", Fired: "show"},
		{Key: "x", Taken: true, Pending: true},
		{Key: "a", Taken: true, Fired: "show,put"},
	})
}

func TestTheClassesCoverTheAlphabetAndNoMore(t *testing.T) {
	a := seq.NewAlphabet([]string{"a", "b"}, "T")
	tab := compile(t, &seq.Grammar{Name: "g", Scopes: []string{"s"}, Delays: map[string]time.Duration{"T": time.Second},
		Rules: []seq.Rule{rule(a, 1, "s", "a T", "one")}}, a)
	for _, c := range []struct {
		name  string
		given seq.Sym
		then  bool
	}{
		{name: "the first key", given: 0, then: true},
		{name: "the last event", given: 2, then: true},
		{name: "past the alphabet", given: 3, then: false},
	} {
		t.Run(c.name, func(t *testing.T) {
			in := tab.Classes.Of(c.given) != fa.NoClass

			if in != c.then {
				t.Errorf("symbol %d is in a class: %v; want %v", c.given, in, c.then)
			}
		})
	}
}

func TestTheOperandIsTheSpanOfWideKeys(t *testing.T) {
	a := alphabet()
	for _, c := range []struct {
		name  string
		given string
		when  string
		then  *seq.Slice
	}{
		{name: "no wide key", given: "a b", when: "a b", then: nil},
		{name: "a repeat between keys", given: "c d|e+ T", when: "c d d T", then: &seq.Slice{From: 1, To: 0}},
		{name: "a wide key before the last", given: "a|b c", when: "b c", then: &seq.Slice{From: 0, To: -1}},
		{name: "two wide keys apart", given: "a|b c d|e x", when: "a c e x", then: &seq.Slice{From: 0, To: -1}},
	} {
		t.Run(c.name, func(t *testing.T) {
			tab := compile(t, &seq.Grammar{Name: "g", Scopes: []string{"s"}, Delays: map[string]time.Duration{"T": time.Second},
				Rules: []seq.Rule{rule(a, 1, "s", c.given, "x")}}, a)
			m := seqtest.Machine(t, tab, "")
			keys := strings.Fields(c.when)
			seqtest.Run(t, m, a, nil, seqtest.Prefix(strings.Join(keys[:len(keys)-1], " ")))
			last, _ := a.Of(keys[len(keys)-1])

			_, fired := m.Step(last, nil)

			if len(fired) != 1 || !reflect.DeepEqual(fired[0].Operand, c.then) {
				t.Fatalf("%s fired %+v; want one action of operand %+v", c.when, fired, c.then)
			}
		})
	}
}

func goes(r seq.Rule, to, arg string) seq.Rule {
	r.Goto, r.Arg = to, arg
	return r
}

func TestAnAmbiguousGrammarHasConflicts(t *testing.T) {
	a := alphabet()
	for _, c := range []struct {
		name  string
		given []seq.Rule
		then  []string
	}{
		{
			name:  "two rules of one key",
			given: []seq.Rule{rule(a, 1, "s", "a", "x"), rule(a, 2, "s", "a", "y")},
			then:  []string{`g:2: "a" and "a" both finish here -- two rules, one key sequence`},
		},
		{
			name:  "a key that another rule goes on from",
			given: []seq.Rule{rule(a, 1, "s", "b", "x"), rule(a, 2, "s", "b c", "y")},
			then:  []string{`g:1: "b" is complete, and b c can still follow -- end it with a delay, or peek at the key after it`},
		},
		{
			name:  "a key that a delay can follow",
			given: []seq.Rule{rule(a, 1, "s", "a", "x"), rule(a, 2, "s", "a T", "y")},
			then:  []string{`g:1: "a" is complete, and a T can still follow -- end it with a delay, or peek at the key after it`},
		},
		{
			name:  "a repeat at the end",
			given: []seq.Rule{rule(a, 1, "s", "a+", "x")},
			then:  []string{`g:1: "a+" is complete, and a+ can still follow -- end it with a delay, or peek at the key after it`},
		},
		{
			name:  "both kinds at once",
			given: []seq.Rule{rule(a, 1, "s", "a", "x"), rule(a, 2, "s", "a", "y"), rule(a, 3, "s", "a b", "z")},
			then: []string{
				`g:1: "a" is complete, and a b can still follow -- end it with a delay, or peek at the key after it`,
				`g:2: "a" and "a" both finish here -- two rules, one key sequence`,
			},
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			g := &seq.Grammar{Name: "g", Scopes: []string{"s"}, Delays: map[string]time.Duration{"T": time.Second}, Rules: c.given}

			tab, err := seq.Compile(g, a)

			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(tab.Conflicts, c.then) {
				t.Errorf("the conflicts are %q; want %q", tab.Conflicts, c.then)
			}
		})
	}
}

func TestABadGrammarDoesNotCompile(t *testing.T) {
	a := alphabet()
	x := func(line int, scope, pattern string) seq.Rule { return rule(a, line, scope, pattern, "x") }
	ahead := func(n int) seq.Rule { r := x(1, "s", "a"); r.Ahead = n; return r }
	nope := []seq.Class{{Event: "nope", Text: "nope"}}
	for _, c := range []struct {
		name  string
		given *seq.Grammar
		then  string
	}{
		{
			name:  "no scopes",
			given: &seq.Grammar{Name: "g"},
			then:  "g: no scopes, so there is nowhere to begin",
		},
		{
			name:  "a scope called this",
			given: &seq.Grammar{Name: "g", Scopes: []string{"this"}},
			then:  "g: a scope cannot be called this, which names the scope a rule is in",
		},
		{
			name:  "a scope declared twice",
			given: &seq.Grammar{Name: "g", Scopes: []string{"s", "s"}},
			then:  "g: scope s is declared twice",
		},
		{
			name:  "a first scope that takes a parameter",
			given: &seq.Grammar{Name: "g", Scopes: []string{"w"}, Params: map[string]string{"w": "then"}},
			then:  "g: w takes a parameter, and the first scope is where a machine begins",
		},
		{
			name:  "a rule in no scope",
			given: &seq.Grammar{Name: "g", Scopes: []string{"s"}, Rules: []seq.Rule{x(1, "u", "a")}},
			then:  "g:1: the rule is in u, and there is no such scope",
		},
		{
			name:  "a goto to no scope",
			given: &seq.Grammar{Name: "g", Scopes: []string{"s"}, Rules: []seq.Rule{goes(x(1, "s", "a"), "typo", "")}},
			then:  "g:1: -> typo, and there is no such scope",
		},
		{
			name: "a parameter given a parameter",
			given: &seq.Grammar{Name: "g", Scopes: []string{"s", "w"}, Params: map[string]string{"w": "then"},
				Rules: []seq.Rule{goes(x(1, "s", "a"), "w", "s"), goes(x(2, "w", "a"), "then", "s")}},
			then: "g:2: then is w's parameter and cannot take one of its own",
		},
		{
			name: "a parameter left out",
			given: &seq.Grammar{Name: "g", Scopes: []string{"s", "w"}, Params: map[string]string{"w": "then"},
				Rules: []seq.Rule{goes(x(1, "s", "a"), "w", "")}},
			then: "g:1: w takes a parameter and was given none",
		},
		{
			name:  "a parameter given to a scope without one",
			given: &seq.Grammar{Name: "g", Scopes: []string{"s"}, Rules: []seq.Rule{goes(x(1, "s", "a"), "s", "s")}},
			then:  "g:1: s takes no parameter and was given s",
		},
		{
			name: "a parameter that is no scope",
			given: &seq.Grammar{Name: "g", Scopes: []string{"s", "w"}, Params: map[string]string{"w": "then"},
				Rules: []seq.Rule{goes(x(1, "s", "a"), "w", "u")}},
			then: "g:1: -> w(u), and there is no scope u",
		},
		{
			name: "a parameter that takes a parameter",
			given: &seq.Grammar{Name: "g", Scopes: []string{"s", "w", "v"}, Params: map[string]string{"w": "then", "v": "then"},
				Rules: []seq.Rule{goes(x(1, "s", "a"), "w", "v")}},
			then: "g:1: -> w(v), and v takes a parameter of its own",
		},
		{
			name:  "a scope read in that is not there",
			given: &seq.Grammar{Name: "g", Scopes: []string{"s"}, Rules: []seq.Rule{{Scope: "s", Include: "nowhere", Line: 1}}},
			then:  "g:1: -> nowhere, and there is no such scope",
		},
		{
			name: "a scope read in that takes a parameter",
			given: &seq.Grammar{Name: "g", Scopes: []string{"s", "w"}, Params: map[string]string{"w": "then"},
				Rules: []seq.Rule{{Scope: "s", Include: "w", Line: 1}}},
			then: "g:1: -> w reads w's rules here, and w takes a parameter",
		},
		{
			name: "a scope read in with an action",
			given: &seq.Grammar{Name: "g", Scopes: []string{"s", "t"},
				Rules: []seq.Rule{{Scope: "s", Include: "t", Action: "x", Line: 1}, x(2, "t", "a")}},
			then: "g:1: -> t with no pattern reads t's rules, and does nothing else",
		},
		{
			name: "scopes read into each other",
			given: &seq.Grammar{Name: "g", Scopes: []string{"s", "t"},
				Rules: []seq.Rule{{Scope: "s", Include: "t", Line: 1}, {Scope: "t", Include: "s", Line: 2}}},
			then: "g:2: -> s goes round in a loop back to t",
		},
		{
			name:  "a rule with no pattern",
			given: &seq.Grammar{Name: "g", Scopes: []string{"s"}, Rules: []seq.Rule{{Scope: "s", Action: "x", Line: 1}}},
			then:  "g:1: a rule with no pattern",
		},
		{
			name:  "a rule that peeks at all its keys",
			given: &seq.Grammar{Name: "g", Scopes: []string{"s"}, Rules: []seq.Rule{ahead(1)}},
			then:  `g:1: "a" reads ahead of every key it has, and would never consume one`,
		},
		{
			name:  "a rule that peeks behind",
			given: &seq.Grammar{Name: "g", Scopes: []string{"s"}, Rules: []seq.Rule{ahead(-1)}},
			then:  `g:1: "a" reads ahead of every key it has, and would never consume one`,
		},
		{
			name: "a wildcard to no scope",
			given: &seq.Grammar{Name: "g", Scopes: []string{"s"},
				Any: []seq.Any{{Scope: "s", Pattern: "a", Elem: elems(a, "a")[0], Goto: "nowhere", Line: 1}}},
			then: "g:1: -> nowhere, and there is no such scope",
		},
		{
			name:  "an event nothing declares",
			given: &seq.Grammar{Name: "g", Scopes: []string{"s"}, Rules: []seq.Rule{{Scope: "s", Pattern: "nope", Elems: nope, Action: "x", Line: 1}}},
			then:  `g:1: "nope" is in a pattern and nothing declares it`,
		},
		{
			name:  "a wildcard on an event nothing declares",
			given: &seq.Grammar{Name: "g", Scopes: []string{"s"}, Any: []seq.Any{{Scope: "s", Pattern: "nope", Elem: nope[0], Action: "x", Line: 1}}},
			then:  `g:1: "nope" is in a pattern and nothing declares it`,
		},
		{
			name:  "a wildcard that repeats",
			given: &seq.Grammar{Name: "g", Scopes: []string{"s"}, Any: []seq.Any{{Scope: "s", Pattern: "a+", Elem: elems(a, "a+")[0], Action: "x", Line: 1}}},
			then:  "g:1: `... a+` repeats, and a wildcard reads one key",
		},
		{
			name: "a template that holds itself",
			given: &seq.Grammar{Name: "g", Scopes: []string{"start", "t"}, Params: map[string]string{"t": "then"},
				Rules: []seq.Rule{goes(x(1, "start", "a"), "t", "start"), goes(x(2, "t", "a"), "t", "this"), goes(x(3, "t", "b"), "then", "")}},
			then: "g: t(t(start)) holds t inside itself, and a scope has no stack to come back by",
		},
		{
			name: "a template that holds itself by a wildcard",
			given: &seq.Grammar{Name: "g", Scopes: []string{"start", "t"}, Params: map[string]string{"t": "then"},
				Rules: []seq.Rule{goes(x(1, "start", "a"), "t", "start"), goes(x(2, "t", "b"), "then", "")},
				Any:   []seq.Any{{Scope: "t", Pattern: "a", Elem: elems(a, "a")[0], Goto: "t", Arg: "this", Line: 3}}},
			then: "g: t(t(start)) holds t inside itself, and a scope has no stack to come back by",
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			_, err := seq.Compile(c.given, a)

			if err == nil || err.Error() != c.then {
				t.Errorf("Compile = %v; want %s", err, c.then)
			}
		})
	}
}

func cond(t *testing.T, op seq.Op, value string) seq.Cond {
	t.Helper()
	c, err := seq.NewCompare("app", op, value)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestACondHoldsByItsEnv(t *testing.T) {
	x, y := seq.Name("x"), seq.Name("y")
	for _, c := range []struct {
		name  string
		given seq.Cond
		when  seq.Env
		then  bool
	}{
		{name: "no condition", given: nil, when: seq.Env{}, then: true},
		{name: "a name that is set", given: x, when: seq.Env{"x": "1"}, then: true},
		{name: "a name that is empty", given: x, when: seq.Env{"x": ""}, then: false},
		{name: "a name that is not set", given: x, when: seq.Env{}, then: false},
		{name: "not a name that is not set", given: seq.Not{Of: x}, when: seq.Env{}, then: true},
		{name: "and with one name set", given: seq.And{x, y}, when: seq.Env{"x": "1"}, then: false},
		{name: "and with both names set", given: seq.And{x, y}, when: seq.Env{"x": "1", "y": "1"}, then: true},
		{name: "or with one name set", given: seq.Or{x, y}, when: seq.Env{"y": "1"}, then: true},
		{name: "or with neither name set", given: seq.Or{x, y}, when: seq.Env{}, then: false},
		{name: "an equal value", given: cond(t, seq.Eq, "c"), when: seq.Env{"app": "c"}, then: true},
		{name: "an unequal value", given: cond(t, seq.Eq, "c"), when: seq.Env{"app": "d"}, then: false},
		{name: "not equal to an equal value", given: cond(t, seq.Ne, "c"), when: seq.Env{"app": "c"}, then: false},
		{name: "not equal to an unequal value", given: cond(t, seq.Ne, "c"), when: seq.Env{"app": "d"}, then: true},
		{name: "a value the pattern matches", given: cond(t, seq.Match, "^(a|b)$"), when: seq.Env{"app": "a"}, then: true},
		{name: "a value the pattern does not match", given: cond(t, seq.Match, "^(a|b)$"), when: seq.Env{"app": "ab"}, then: false},
		{name: "the zero comparison", given: &seq.Compare{}, when: seq.Env{"": ""}, then: false},
	} {
		t.Run(c.name, func(t *testing.T) {
			if got := seq.Holds(c.given, c.when); got != c.then {
				t.Errorf("Holds(%v, %v) = %v; want %v", c.given, c.when, got, c.then)
			}
		})
	}
}

func show(c seq.Cond) string {
	if c == nil {
		return "nothing"
	}
	return c.String()
}

func TestACondReadsAsWritten(t *testing.T) {
	x, y := seq.Name("x"), seq.Name("y")
	for _, c := range []struct {
		name  string
		given seq.Cond
		then  string
	}{
		{name: "a name", given: x, then: "x"},
		{name: "not a name", given: seq.Not{Of: x}, then: "not x"},
		{name: "not of an and", given: seq.Not{Of: seq.And{x, y}}, then: "not (x and y)"},
		{name: "an or of an and", given: seq.Or{seq.And{x, seq.Not{Of: cond(t, seq.Match, "^(a|b)$")}}, cond(t, seq.Eq, "c")}, then: `(x and not app ~ "^(a|b)$") or app == "c"`},
		{name: "an and of an or", given: seq.And{seq.Or{x, y}, cond(t, seq.Ne, "c")}, then: `(x or y) and app != "c"`},
	} {
		t.Run(c.name, func(t *testing.T) {
			if got := c.given.String(); got != c.then {
				t.Errorf("String() = %q; want %q", got, c.then)
			}
		})
	}
}

func TestAnOpReadsAsWritten(t *testing.T) {
	for _, c := range []struct {
		name  string
		given seq.Op
		then  string
	}{
		{name: "equal", given: seq.Eq, then: "=="},
		{name: "not equal", given: seq.Ne, then: "!="},
		{name: "match", given: seq.Match, then: "~"},
		{name: "no op", given: seq.NoOp, then: "?"},
	} {
		t.Run(c.name, func(t *testing.T) {
			if got := c.given.String(); got != c.then {
				t.Errorf("String() = %q; want %q", got, c.then)
			}
		})
	}
}

func TestACondNamesWhatItReadsOnce(t *testing.T) {
	given := seq.Or{seq.And{seq.Name("y"), seq.Not{Of: seq.Name("x")}}, cond(t, seq.Eq, "c"), seq.Name("x")}

	names := given.Names()

	if want := []string{"app", "x", "y"}; !reflect.DeepEqual(names, want) {
		t.Errorf("Names() = %q; want %q", names, want)
	}
}

func TestACompareNeedsAnOpAndAPattern(t *testing.T) {
	for _, c := range []struct {
		name  string
		given seq.Op
		when  string
		then  string
	}{
		{name: "no op", given: seq.NoOp, when: "c", then: "? is not a comparison"},
		{name: "a pattern that does not compile", given: seq.Match, when: "(", then: "error parsing regexp: missing closing ): `(`"},
	} {
		t.Run(c.name, func(t *testing.T) {
			_, err := seq.NewCompare("app", c.given, c.when)

			if err == nil || err.Error() != c.then {
				t.Errorf("NewCompare = %v; want %s", err, c.then)
			}
		})
	}
}

func TestAllOfAndAnyOfJoinConditions(t *testing.T) {
	x, y, z := seq.Name("x"), seq.Name("y"), seq.Name("z")
	for _, c := range []struct {
		name  string
		given func(a, b seq.Cond) seq.Cond
		when  [2]seq.Cond
		then  string
	}{
		{name: "all of nothing", given: seq.AllOf, when: [2]seq.Cond{nil, nil}, then: "nothing"},
		{name: "all of nothing and a name", given: seq.AllOf, when: [2]seq.Cond{nil, x}, then: "x"},
		{name: "all of a name and nothing", given: seq.AllOf, when: [2]seq.Cond{x, nil}, then: "x"},
		{name: "all of an and and nothing", given: seq.AllOf, when: [2]seq.Cond{seq.And{x, y}, nil}, then: "x and y"},
		{name: "all of an and and a name", given: seq.AllOf, when: [2]seq.Cond{seq.And{x, y}, z}, then: "x and y and z"},
		{name: "all of two names", given: seq.AllOf, when: [2]seq.Cond{x, y}, then: "x and y"},
		{name: "all of an or and a name", given: seq.AllOf, when: [2]seq.Cond{seq.Or{x, y}, z}, then: "(x or y) and z"},
		{name: "any of two names", given: seq.AnyOf, when: [2]seq.Cond{x, y}, then: "x or y"},
		{name: "any of an or and a name", given: seq.AnyOf, when: [2]seq.Cond{seq.Or{x, y}, z}, then: "x or y or z"},
		{name: "any of an and and a name", given: seq.AnyOf, when: [2]seq.Cond{seq.And{x, y}, z}, then: "(x and y) or z"},
	} {
		t.Run(c.name, func(t *testing.T) {
			if got := show(c.given(c.when[0], c.when[1])); got != c.then {
				t.Errorf("got %s; want %s", got, c.then)
			}
		})
	}
}

func TestAllOfAndAnyOfLeaveWhatTheyExtendAlone(t *testing.T) {
	and, or := make(seq.And, 2, 4), make(seq.Or, 2, 4)
	and[0], and[1] = seq.Name("x"), seq.Name("y")
	or[0], or[1] = seq.Name("x"), seq.Name("y")

	all, anyOf := seq.AllOf(and, seq.Name("z")), seq.AnyOf(or, seq.Name("z"))
	seq.AllOf(and, seq.Name("w"))
	seq.AnyOf(or, seq.Name("w"))

	if all.String() != "x and y and z" || anyOf.String() != "x or y or z" {
		t.Errorf("after joining w, the earlier joins read %q and %q", all, anyOf)
	}
}

func TestAScopeOnlyAWildcardGoesToIsCompiled(t *testing.T) {
	a := alphabet()
	g := &seq.Grammar{Name: "g", Scopes: []string{"start", "mode"},
		Rules: []seq.Rule{rule(a, 1, "start", "a", "one"), rule(a, 2, "mode", "x", "ex")},
		Any:   []seq.Any{{Scope: "start", Pattern: "m", Elem: elems(a, "m")[0], Goto: "mode", Line: 3}}}
	tab := compile(t, g, a)
	m := seqtest.Machine(t, tab, "")

	seqtest.Run(t, m, a, nil, []seqtest.Step{
		{Key: "m", Taken: true},
		{Key: "x", Taken: true, Fired: "ex"},
	})

	if names := scopeNames(tab); names != "start mode" {
		t.Errorf("the scopes are %q; want start mode", names)
	}
}

func TestAMachineNeedsATableWithAScope(t *testing.T) {
	given := &seq.Table{Name: "empty"}

	_, err := seq.NewMachine(given, "")

	if want := `empty has no scope ""`; err == nil || err.Error() != want {
		t.Errorf("NewMachine = %v; want %s", err, want)
	}
}

func TestAnyReadsKeysAndNotDelays(t *testing.T) {
	a := alphabet()
	tab := compile(t, &seq.Grammar{Name: "g", Scopes: []string{"s"}, Delays: map[string]time.Duration{"T": time.Second},
		Rules: []seq.Rule{rule(a, 1, "s", "x any", "other")}}, a)
	m := seqtest.Machine(t, tab, "")

	seqtest.Run(t, m, a, nil, []seqtest.Step{
		{Key: "x", Taken: true, Pending: true},
		{Key: "T", Pending: true},
		{Key: "a", Taken: true, Fired: "other"},
	})
}
