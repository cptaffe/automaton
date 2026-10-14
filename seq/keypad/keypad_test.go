package keypad

import (
	"reflect"
	"strings"
	"testing"

	"github.com/cptaffe/automaton/seq"
	"github.com/cptaffe/automaton/seq/seqtest"
)

const plan = `%delay T 5s
%delay abandon 16s

start:
  *98           do voicemail  -> mailbox
  0             do operator
  NXX-XXXX T    do line
  960-XXX-XXXX  do line-ten
  ... abandon   do dial-again

mailbox:
  1             do play       when holding
  #             do stop       -> start
`

func parse(t *testing.T, src, from string) *seq.Grammar {
	t.Helper()
	g, err := Parse(strings.NewReader(src), from)
	if err != nil {
		t.Fatal(err)
	}
	return g
}

func compile(t *testing.T, trees ...*seq.Grammar) *seq.Table {
	t.Helper()
	tab, err := Compile(trees...)
	if err != nil {
		t.Fatal(err)
	}
	if len(tab.Conflicts) > 0 {
		t.Fatal(tab.Conflicts)
	}
	return tab
}

func then(prefix string, last seqtest.Step) []seqtest.Step {
	return append(seqtest.Prefix(prefix), last)
}

var holding = seq.Env{"holding": "yes"}

func TestAPlanAnswersWhatIsDialled(t *testing.T) {
	tab := compile(t, parse(t, plan, "plan.tree"))
	for _, c := range []struct {
		name  string
		given string
		when  []seqtest.Step
		then  string
	}{
		{
			name:  "a code that goes to another scope",
			given: "start",
			when:  then("* 9", seqtest.Step{Key: "8", Taken: true, Fired: "voicemail"}),
			then:  "mailbox",
		},
		{
			name:  "one key",
			given: "start",
			when:  []seqtest.Step{{Key: "0", Taken: true, Fired: "operator"}},
			then:  "start",
		},
		{
			name:  "seven digits and a pause",
			given: "start",
			when:  then("4 8 2 1 2 3 4", seqtest.Step{Key: "T", Taken: true, Fired: "line"}),
			then:  "start",
		},
		{
			name:  "seven digits that begin as ten and a pause",
			given: "start",
			when:  then("9 6 0 1 2 3 4", seqtest.Step{Key: "T", Taken: true, Fired: "line"}),
			then:  "start",
		},
		{
			name:  "ten digits",
			given: "start",
			when:  then("9 6 0 4 8 2 1 2 3", seqtest.Step{Key: "4", Taken: true, Fired: "line-ten"}),
			then:  "start",
		},
		{
			name:  "a number left unfinished",
			given: "start",
			when:  then("4", seqtest.Step{Key: "abandon", Taken: true, Fired: "dial-again"}),
			then:  "start",
		},
		{
			name:  "a pause with nothing dialled",
			given: "start",
			when:  []seqtest.Step{{Key: "T"}},
			then:  "start",
		},
		{
			name:  "a key whose when holds",
			given: "mailbox",
			when:  []seqtest.Step{{Key: "1", Taken: true, Fired: "play"}},
			then:  "mailbox",
		},
		{
			name:  "a key whose when does not hold",
			given: "mailbox",
			when:  []seqtest.Step{{Key: "1", Env: seq.Env{}}},
			then:  "mailbox",
		},
		{
			name:  "a key that goes back",
			given: "mailbox",
			when:  []seqtest.Step{{Key: "#", Taken: true, Fired: "stop"}},
			then:  "start",
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			m := seqtest.Machine(t, tab, c.given)

			seqtest.Run(t, m, tab.Alpha, holding, c.when)

			if m.Scope() != c.then {
				t.Errorf("the machine is in %s; want %s", m.Scope(), c.then)
			}
		})
	}
}

func TestAPlanWaitsForTheDelaysItReads(t *testing.T) {
	given := parse(t, plan, "plan.tree")

	tab := compile(t, given)

	if got := tab.Waiting(); !reflect.DeepEqual(got, []string{"T", "abandon"}) {
		t.Errorf("the plan waits for %q; want T and abandon", got)
	}
}

func TestTheOperandIsTheNumberDialled(t *testing.T) {
	tab := compile(t, parse(t, plan, "plan.tree"))
	for _, c := range []struct {
		name  string
		given string
		then  *seq.Slice
	}{
		{name: "a code", given: "* 9 8", then: nil},
		{name: "seven digits", given: "4 8 2 1 2 3 4 T", then: &seq.Slice{From: 0, To: 0}},
		{name: "ten digits", given: "9 6 0 4 8 2 1 2 3 4", then: &seq.Slice{From: 3, To: 0}},
	} {
		t.Run(c.name, func(t *testing.T) {
			m := seqtest.Machine(t, tab, "")
			keys := strings.Fields(c.given)
			seqtest.Run(t, m, tab.Alpha, nil, seqtest.Prefix(strings.Join(keys[:len(keys)-1], " ")))
			last, _ := tab.Alpha.Of(keys[len(keys)-1])

			_, fired := m.Step(last, nil)

			if len(fired) != 1 || !reflect.DeepEqual(fired[0].Operand, c.then) {
				t.Errorf("%s fired %+v; want one action of operand %+v", c.given, fired, c.then)
			}
		})
	}
}

const dialTree = `%delay T 1s
start:
  0     do operator  -> dialing(menu)
  1     do one
  NXX   do line  -> connected
  ... T  do idle

dialing(then):
  answered  -> then

connected:
`

func TestATreeHandsOff(t *testing.T) {
	tab := compile(t, parse(t, dialTree, "dial.tree"), parse(t, "start:\n  1  do menu-one\n", "menu.tree"))
	for _, c := range []struct {
		name string
		when []seqtest.Step
		then string
	}{
		{
			name: "to a scope that waits",
			when: []seqtest.Step{{Key: "0", Taken: true, Fired: "operator"}, {Key: "1"}, {Key: "T"}},
			then: "dial.dialing(menu.start)",
		},
		{
			name: "to another tree",
			when: []seqtest.Step{{Key: "0", Taken: true, Fired: "operator"}, {Key: "answered", Taken: true}, {Key: "1", Taken: true, Fired: "menu-one"}},
			then: "menu.start",
		},
		{
			name: "to a scope of its own",
			when: then("2 3", seqtest.Step{Key: "4", Taken: true, Fired: "line"}),
			then: "dial.connected",
		},
		{
			name: "nowhere",
			when: []seqtest.Step{{Key: "T", Taken: true, Fired: "idle"}},
			then: "dial.start",
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			m := seqtest.Machine(t, tab, "")

			seqtest.Run(t, m, tab.Alpha, nil, c.when)

			if m.Scope() != c.then {
				t.Errorf("the machine is in %s; want %s", m.Scope(), c.then)
			}
		})
	}
}

type linked struct {
	scope, include, to, arg string
}

func TestLinkNamesEachScopeByItsTree(t *testing.T) {
	dial := parse(t, "start:\n  0  -> wait(this)\n  1  -> wait(menu)\n  2  -> menu\n\nwait(then):\n  answered  -> then\n  ... 3  -> then\n  ... 4  -> start\n", "dial.tree")
	dial.Rules = append(dial.Rules, seq.Rule{Scope: "start", Include: "extra", Line: 9})
	dial.Scopes = append(dial.Scopes, "extra")
	menu := parse(t, "start:\n  1  do one\n", "menu.tree")

	g, err := Link(dial, menu)

	if err != nil {
		t.Fatal(err)
	}
	got := make([]linked, 0, len(g.Rules)+len(g.Any))
	for _, r := range g.Rules {
		got = append(got, linked{r.Scope, r.Include, r.Goto, r.Arg})
	}
	for _, x := range g.Any {
		got = append(got, linked{x.Scope, "", x.Goto, x.Arg})
	}
	want := []linked{
		{"dial.start", "", "dial.wait", "this"},
		{"dial.start", "", "dial.wait", "menu.start"},
		{"dial.start", "", "menu.start", ""},
		{"dial.wait", "", "then", ""},
		{"dial.start", "dial.extra", "", ""},
		{"menu.start", "", "", ""},
		{"dial.wait", "", "then", ""},
		{"dial.wait", "", "dial.start", ""},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("the linked rules are\n%+v\nwant\n%+v", got, want)
	}
	if want := []string{"dial.start", "dial.wait", "dial.extra", "menu.start"}; !reflect.DeepEqual(g.Scopes, want) {
		t.Errorf("the scopes are %q; want %q", g.Scopes, want)
	}
}

func TestLinkSharesADelayTreesAgreeOn(t *testing.T) {
	a := parse(t, "%delay T 1s\nstart:\n  1 T  do one\n", "a.tree")
	b := parse(t, "%delay T 1s\nstart:\n  2 T  do two\n", "b.tree")

	tab := compile(t, a, b)

	if got := tab.Waiting(); !reflect.DeepEqual(got, []string{"T"}) {
		t.Errorf("the trees wait for %q; want T", got)
	}
}

func TestABadPlanDoesNotParse(t *testing.T) {
	for _, c := range []struct {
		name  string
		given string
		then  string
	}{
		{name: "a rule above every scope", given: "1 do a\n", then: "t.tree:1: \"1\" is above every scope, where only a `...` rule can be"},
		{name: "a rule with nowhere to go", given: "s:\n  1\n", then: `t.tree:2: "1" has a pattern and nowhere to go`},
		{name: "an action in capitals", given: "s:\n  1 do Put\n", then: `t.tree:2: the action "Put" is not lower case and hyphens`},
		{name: "an action after a blank line", given: "s:\n  1 do a\n\n  2 do Put\n", then: `t.tree:4: the action "Put" is not lower case and hyphens`},
		{name: "a wildcard's action in capitals", given: "s:\n  ... 1 do Put\n", then: `t.tree:2: the action "Put" is not lower case and hyphens`},
		{name: "a symbol after the action", given: "s:\n  1 do a 2\n", then: `t.tree:2: "2" is a pattern symbol, and one cannot stand here`},
		{name: "a name after the condition", given: "s:\n  1 do a when x y\n", then: `t.tree:2: "y" is a name, and one cannot stand here`},
		{name: "a when with no condition", given: "s:\n  1 do a when\n", then: "t.tree:2: a new line cannot stand here"},
		{name: "a goto with no scope", given: "s:\n  1 do a ->\n", then: "t.tree:2: a new line cannot stand here"},
		{name: "a stray parenthesis", given: "s:\n  1 (\n", then: "t.tree:2: ( cannot stand here"},
		{name: "a declaration that is not one", given: "%bogus\n", then: "t.tree:1: %bogus is not a declaration"},
		{name: "a delay in a scope", given: "s:\n  %delay T 1s\n", then: "t.tree:2: %delay T 1s belongs above every scope: a symbol lasts as long wherever it is read"},
		{name: "a delay that is not a duration", given: "%delay T soon\n", then: "t.tree:1: %delay T soon: soon is not a duration"},
		{name: "a delay of no time", given: "%delay T 0s\n", then: "t.tree:1: %delay T 0s: a wait of no time is not a wait"},
		{name: "a delay of less than no time", given: "%delay T -1s\n", then: "t.tree:1: %delay T -1s: a wait of no time is not a wait"},
		{name: "a delay changed", given: "%delay T 1s\n%delay T 2s\n", then: "t.tree:2: %delay T 2s: it is already 1s"},
		{name: "a delay called any", given: "%delay any 1s\n", then: "t.tree:1: %delay any: any is every key, not a delay"},
		{name: "a wildcard of two symbols", given: "s:\n  ... 1 2 do a\n", then: "t.tree:2: `... 1 2` is more than one symbol"},
		{name: "a template opened again with another parameter", given: "w(x):\n  1 -> x\nw(y):\n", then: "t.tree:3: w is opened again with another parameter"},
		{name: "a scope opened again with a parameter", given: "s:\n  1 do a\ns(x):\n", then: "t.tree:3: s is opened again with another parameter"},
	} {
		t.Run(c.name, func(t *testing.T) {
			_, err := Parse(strings.NewReader(c.given), "t.tree")

			if err == nil || err.Error() != c.then {
				t.Errorf("Parse = %v; want %s", err, c.then)
			}
		})
	}
}

type tree struct{ from, src string }

func TestABadPlanDoesNotCompile(t *testing.T) {
	for _, c := range []struct {
		name  string
		given []tree
		then  string
	}{
		{name: "a goto to no scope", given: []tree{{"t.tree", "s:\n  1 do a -> t\n"}}, then: "t:2: -> t, and there is no such scope"},
		{name: "a parameter given a parameter", given: []tree{{"t.tree", "s:\n  1 do a -> w(s)\nw(x):\n  2 -> x(s)\n"}}, then: "t:4: x is w's parameter and cannot take one of its own"},
		{name: "a parameter left out", given: []tree{{"t.tree", "s:\n  1 do a -> w\nw(x):\n  2 -> x\n"}}, then: "t:2: w takes a parameter and was given none"},
		{name: "a parameter given to a scope without one", given: []tree{{"t.tree", "s:\n  1 do a -> s(s)\n"}}, then: "t:2: s takes no parameter and was given s"},
		{name: "a first scope that takes a parameter", given: []tree{{"t.tree", "w(x):\n  1 -> x\ns:\n  2 -> w(s)\n"}}, then: "t: w takes a parameter, and the first scope is where a machine begins"},
		{name: "a symbol nothing declares", given: []tree{{"t.tree", "s:\n  1 nope do a\n"}}, then: `t:2: "nope" is in a pattern and nothing declares it`},
		{name: "a wildcard that repeats", given: []tree{{"t.tree", "s:\n  ... 1+ do a\n"}}, then: "t:2: `... 1+` repeats, and a wildcard reads one key"},
		{name: "a hand-off to a tree that is not there", given: []tree{{"t.tree", dialTree}}, then: "t:3: -> dialing(menu), and there is no scope menu"},
		{name: "two trees of one name", given: []tree{{"t.tree", "s:\n  1 do a\n"}, {"t/t.tree", "s:\n  2 do b\n"}}, then: "two trees are called t"},
		{name: "a goto to no scope or tree", given: []tree{{"t.tree", "s:\n  1 do a -> v\n"}, {"u.tree", "start:\n  1 do b\n"}}, then: "t:2: -> v, and there is no such scope or tree"},
		{name: "a parameter that is no scope or tree", given: []tree{{"t.tree", "s:\n  1 do a -> w(v)\nw(x):\n  2 -> x\n"}, {"u.tree", "start:\n  1 do b\n"}}, then: "t:2: -> v, and there is no such scope or tree"},
		{name: "a wildcard to no scope or tree", given: []tree{{"t.tree", "s:\n  ... 1 do a -> v\n"}, {"u.tree", "start:\n  1 do b\n"}}, then: "t:2: -> v, and there is no such scope or tree"},
		{name: "a wildcard's parameter that is no scope or tree", given: []tree{{"t.tree", "s:\n  ... 1 do a -> w(v)\nw(x):\n  2 -> x\n"}, {"u.tree", "start:\n  1 do b\n"}}, then: "t:2: -> v, and there is no such scope or tree"},
		{name: "two trees that disagree on a delay", given: []tree{{"t.tree", "%delay T 1s\ns:\n  1 T do a\n"}, {"u.tree", "%delay T 2s\nstart:\n  2 T do b\n"}}, then: "u waits 2s for T, and another tree 1s"},
	} {
		t.Run(c.name, func(t *testing.T) {
			trees := make([]*seq.Grammar, 0, len(c.given))
			for _, x := range c.given {
				trees = append(trees, parse(t, x.src, x.from))
			}

			_, err := Compile(trees...)

			if err == nil || err.Error() != c.then {
				t.Errorf("Compile = %v; want %s", err, c.then)
			}
		})
	}
}

func TestAnAmbiguousPlanHasConflicts(t *testing.T) {
	for _, c := range []struct {
		name  string
		given string
		then  []string
	}{
		{name: "two rules of one key", given: "s:\n  1 do a\n  1 do b\n", then: []string{`t:3: "1" and "1" both finish here -- two rules, one key sequence`}},
		{name: "a key another rule goes on from", given: "s:\n  1 do a\n  12 do b\n", then: []string{`t:2: "1" is complete, and 12 can still follow -- end it with a delay, or peek at the key after it`}},
		{name: "a repeat at the end", given: "s:\n  1X+ do a\n", then: []string{`t:2: "1X+" is complete, and 1X+ can still follow -- end it with a delay, or peek at the key after it`}},
	} {
		t.Run(c.name, func(t *testing.T) {
			tab, err := Compile(parse(t, c.given, "t.tree"))

			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(tab.Conflicts, c.then) {
				t.Errorf("the conflicts are %q; want %q", tab.Conflicts, c.then)
			}
		})
	}
}

type commented struct{ note, label string }

func TestCommentsGiveTheNoteAndLabels(t *testing.T) {
	for _, c := range []struct {
		name  string
		given string
		then  commented
	}{
		{name: "a comment above every scope", given: "; dial plan\nstart:\n  0  do operator\n", then: commented{"dial plan", ""}},
		{name: "only the first comment", given: "; dial plan\n; more\nstart:\n  0  do operator\n", then: commented{"dial plan", ""}},
		{name: "a blank comment before the note", given: ";\n; dial plan\nstart:\n  0  do operator\n", then: commented{"dial plan", ""}},
		{name: "no comment above every scope", given: "start:\n  ; inside\n  0  do operator\n", then: commented{"", ""}},
		{name: "a label", given: "start:\n  ; label: the operator\n  0  do operator\n", then: commented{"", "the operator"}},
		{name: "a label among remarks", given: "start:\n  ; a remark\n  ; label: the operator\n  ; another\n  0  do operator\n", then: commented{"", "the operator"}},
		{name: "a label cut off by a blank line", given: "start:\n  ; label: the operator\n\n  0  do operator\n", then: commented{"", ""}},
		{name: "a label at the end of a line", given: "start:\n  1  do one  ; label: one\n  0  do operator\n", then: commented{"", ""}},
		{name: "a label above a scope", given: "; label: start\nstart:\n  0  do operator\n", then: commented{"label: start", ""}},
	} {
		t.Run(c.name, func(t *testing.T) {
			g := parse(t, c.given, "p.tree")

			got := commented{g.Note, g.Rules[len(g.Rules)-1].Label}

			if got != c.then {
				t.Errorf("the note and label are %+v; want %+v", got, c.then)
			}
		})
	}
}

func TestAWhenReadsAsWritten(t *testing.T) {
	given := "start:\n  1  do one  when a and (b or not c)\n"

	g := parse(t, given, "p.tree")

	if got := g.Rules[0].When.String(); got != "a and (b or not c)" {
		t.Errorf("when %q", got)
	}
}

func TestAnActionIsLowerCaseAndHyphens(t *testing.T) {
	for _, c := range []struct {
		name  string
		given string
		then  string
	}{
		{name: "the first letter", given: "a", then: ""},
		{name: "the last letter", given: "zap", then: ""},
		{name: "hyphens", given: "dial-again", then: ""},
		{name: "a capital", given: "Put", then: `the action "Put" is not lower case and hyphens`},
		{name: "a digit", given: "line2", then: `the action "line2" is not lower case and hyphens`},
		{name: "a brace", given: "a{", then: `the action "a{" is not lower case and hyphens`},
		{name: "a backquote", given: "a`", then: "the action \"a`\" is not lower case and hyphens"},
	} {
		t.Run(c.name, func(t *testing.T) {
			err := named(c.given)

			if got := errorText(err); got != c.then {
				t.Errorf("named(%q) = %q; want %q", c.given, got, c.then)
			}
		})
	}
}

func errorText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func TestAScopeMayBeOpenedAgain(t *testing.T) {
	for _, c := range []struct {
		name  string
		given string
		then  []string
	}{
		{name: "a scope", given: "s:\n  1 do a\nt:\n  2 do b\ns:\n  3 do c\n", then: []string{"s", "t"}},
		{name: "a template with its parameter", given: "s:\n  1 -> w(s)\nw(x):\n  2 -> x\nw(x):\n  3 -> x\n", then: []string{"s", "w"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			g := parse(t, c.given, "t.tree")

			if !reflect.DeepEqual(g.Scopes, c.then) {
				t.Errorf("the scopes are %q; want %q", g.Scopes, c.then)
			}
		})
	}
}

func TestAPatternReadsAsWritten(t *testing.T) {
	given := "start:\n  960-NXX-XXXX  do far\n  1  answered   do up\n  ... abandon   do again\n"

	tree := parse(t, given, "p.tree")

	got := make([]string, 0, len(tree.Rules)+len(tree.Any))
	for _, r := range tree.Rules {
		got = append(got, r.Pattern)
	}
	for _, x := range tree.Any {
		got = append(got, x.Pattern)
	}
	if want := []string{"960-NXX-XXXX", "1 answered", "abandon"}; !reflect.DeepEqual(got, want) {
		t.Errorf("patterns %q; want %q", got, want)
	}
}

func TestAStrayKeyIsThePlans(t *testing.T) {
	tab := compile(t, parse(t, "start:\n  NXX  do local\n  ... any  do reorder\n", "p.tree"))
	for _, c := range []struct {
		name string
		when []seqtest.Step
	}{
		{name: "a number", when: then("9 6", seqtest.Step{Key: "0", Taken: true, Fired: "local"})},
		{name: "a key that cuts a number off", when: then("9", seqtest.Step{Key: "*", Taken: true, Fired: "reorder"})},
		{name: "a key that begins no number", when: []seqtest.Step{{Key: "#", Taken: true, Fired: "reorder"}}},
	} {
		t.Run(c.name, func(t *testing.T) {
			seqtest.Run(t, seqtest.Machine(t, tab, ""), tab.Alpha, nil, c.when)
		})
	}
}
