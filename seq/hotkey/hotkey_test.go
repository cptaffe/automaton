package hotkey

import (
	"reflect"
	"strings"
	"testing"

	"github.com/cptaffe/automaton/seq"
	"github.com/cptaffe/automaton/seq/seqtest"
)

const keytab = `# Edit ,>9p write acme-hotkey/rules
%{
fn put { echo put | 9p write acme/$winid/ctl }
%}
%{
fn del { echo del | 9p write acme/$winid/ctl }
%}
%delay settle 1s

main:
	-> acme when app == "devdraw" || app ~ "^9term$"

acme:
	command-s            { put }                 # save
	shift-command-b      { Lrefs }               # references
	control-x control-s  { put }
	control-x settle     { echo '}' {nested} }   # look
	control-x peek any   { echo '}' {nested} }   # look
	command--            { Shrink }
	command-m            -> mode

mode:
	j                    { down }
	escape               { exit } -> main
`

func parse(t *testing.T, src string) *Rules {
	t.Helper()
	r, err := Parse(strings.NewReader(src), "hotkeys", nil)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

type parsed struct {
	scope, pattern, include, action, to, label string
}

func TestParseReadsEachRule(t *testing.T) {
	given := keytab

	r := parse(t, given)

	got := make([]parsed, 0, len(r.Grammar.Rules))
	for _, x := range r.Grammar.Rules {
		got = append(got, parsed{x.Scope, x.Pattern, x.Include, x.Action, x.Goto, x.Label})
	}
	want := []parsed{
		{"main", "", "acme", "", "", ""},
		{"acme", "command-s", "", "put", "", "save"},
		{"acme", "shift-command-b", "", "Lrefs", "", "references"},
		{"acme", "control-x control-s", "", "put", "", ""},
		{"acme", "control-x settle", "", "echo '}' {nested}", "", "look"},
		{"acme", "control-x peek any", "", "echo '}' {nested}", "", "look"},
		{"acme", "command--", "", "Shrink", "", ""},
		{"acme", "command-m", "", "", "mode", ""},
		{"mode", "j", "", "down", "", ""},
		{"mode", "escape", "", "exit", "main", ""},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("the rules are\n%+v\nwant\n%+v", got, want)
	}
	if when := r.Grammar.Rules[0].When.String(); when != `app == "devdraw" or app ~ "^9term$"` {
		t.Errorf("the include holds when %s", when)
	}
}

func TestParseJoinsThePreludes(t *testing.T) {
	given := keytab

	r := parse(t, given)

	if want := "fn put { echo put | 9p write acme/$winid/ctl }\nfn del { echo del | 9p write acme/$winid/ctl }"; r.Prelude != want {
		t.Errorf("the prelude is %q; want %q", r.Prelude, want)
	}
}

var (
	acme   = seq.Env{"app": "devdraw"}
	nine   = seq.Env{"app": "9term"}
	safari = seq.Env{"app": "Safari"}
)

func TestAMachineMatchesKeysByApp(t *testing.T) {
	r := parse(t, keytab)
	m := seqtest.Machine(t, r.Table, "")

	seqtest.Run(t, m, r.Table.Alpha, acme, []seqtest.Step{
		{Key: "command-s", Taken: true, Fired: "put"},
		{Key: "command-shift-b", Taken: true, Fired: "Lrefs"},
		{Key: "command--", Taken: true, Fired: "Shrink"},
		{Key: "command-s", Env: nine, Taken: true, Fired: "put"},
		{Key: "command-s", Env: safari},
		{Key: "space"},
		{Key: "control-x", Taken: true, Pending: true},
		{Key: "control-s", Taken: true, Fired: "put"},
		{Key: "control-x", Taken: true, Pending: true},
		{Key: "space", Fired: "echo '}' {nested}"},
		{Key: "control-x", Taken: true, Pending: true},
		{Key: "command-s", Taken: true, Fired: "echo '}' {nested},put"},
		{Key: "control-x", Taken: true, Pending: true},
		{Key: "settle", Taken: true, Fired: "echo '}' {nested}"},
		{Key: "command-m", Taken: true},
		{Key: "j", Env: safari, Taken: true, Fired: "down"},
		{Key: "command-s"},
		{Key: "escape", Taken: true, Fired: "exit"},
		{Key: "j"},
		{Key: "command-s", Env: safari},
		{Key: "command-s", Taken: true, Fired: "put"},
	})
}

func TestAPrefixWaitsForItsDelay(t *testing.T) {
	r := parse(t, keytab)
	for _, c := range []struct {
		name  string
		given string
		then  []string
	}{
		{name: "nothing pressed", given: "", then: nil},
		{name: "a prefix a delay ends", given: "control-x", then: []string{"settle"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			m := seqtest.Machine(t, r.Table, "")
			seqtest.Run(t, m, r.Table.Alpha, acme, seqtest.Prefix(c.given))

			waits := m.Waits()

			if !reflect.DeepEqual(waits, c.then) {
				t.Errorf("after %q the machine waits for %q; want %q", c.given, waits, c.then)
			}
		})
	}
}

func TestABadFileDoesNotParse(t *testing.T) {
	for _, c := range []struct {
		name  string
		given string
		then  string
	}{
		{
			name:  "two rules of one key",
			given: "a:\n\tcommand-s { x }\n\tcommand-s { y }\n",
			then:  `hotkeys:3: "command-s" and "command-s" both finish here -- two rules, one key sequence`,
		},
		{
			name:  "a key that another rule goes on from",
			given: "a:\n\tcontrol-x { x }\n\tcontrol-x control-s { y }\n",
			then:  `hotkeys:2: "control-x" is complete, and control-x control-s can still follow -- end it with a delay, or peek at the key after it`,
		},
		{
			name:  "a key not on the keyboard",
			given: "a:\n\tcommand-zz { x }\n",
			then:  "hotkeys:2: command-zz: no such key",
		},
		{
			name:  "two keys not on the keyboard",
			given: "a:\n\tcommand-zz { x }\n\tcontrol-x yy { y }\n",
			then:  "hotkeys:2: command-zz: no such key\nhotkeys:3: yy: no such key",
		},
		{
			name:  "a wildcard of a key not on the keyboard",
			given: "... command-zz { x }\n",
			then:  "hotkeys:1: command-zz: no such key",
		},
		{
			name:  "a modifier twice",
			given: "a:\n\tcommand-command-s { x }\n",
			then:  "hotkeys:2: command-command-s: command twice",
		},
		{
			name:  "a block left open",
			given: "a:\n\tcommand-s { x\n",
			then:  "hotkeys:3: the input ends inside block",
		},
		{
			name:  "a prelude left open",
			given: "%{\nfn x\n",
			then:  "hotkeys:2: the file ends in the middle of a rule",
		},
		{
			name:  "a rule above every scope",
			given: "command-s { x }\n",
			then:  "hotkeys:1: a rule above every scope; only a ... rule can be there",
		},
		{
			name:  "a rule with nowhere to go",
			given: "a:\n\tcommand-s\n",
			then:  "hotkeys:2: a rule with keys and nowhere to go",
		},
		{
			name:  "a pattern that does not compile",
			given: "a:\n\t-> a when app ~ \"(\"\n",
			then:  "hotkeys:2: error parsing regexp: missing closing ): `(`",
		},
		{
			name:  "a when on a scope",
			given: "a: when app == \"x\"\n",
			then:  "hotkeys:1: when cannot stand here",
		},
		{
			name:  "a stray colon",
			given: "a:\n\tcommand-s { x } :\n",
			then:  "hotkeys:2: : cannot stand here",
		},
		{
			name:  "an include above every scope",
			given: "-> a\na:\n\tcommand-s { x }\n",
			then:  "hotkeys:1: -> a is above every scope, and it reads a's rules into one",
		},
		{
			name:  "scopes read into each other",
			given: "a:\n\t-> b\nb:\n\t-> a\n",
			then:  "hotkeys:4: -> a goes round in a loop back to b",
		},
		{
			name:  "a goto to no scope",
			given: "a:\n\tcommand-s -> b\n",
			then:  "hotkeys:2: -> b, and there is no such scope",
		},
		{
			name:  "an include of no scope",
			given: "a:\n\tcommand-s { x }\n\t-> nowhere\n",
			then:  "hotkeys:3: -> nowhere, and there is no such scope",
		},
		{
			name:  "a delay in a scope",
			given: "a:\n\t%delay x 1s\n",
			then:  "hotkeys:2: %delay x belongs above every scope",
		},
		{
			name:  "a delay that is not a duration",
			given: "%delay x soon\n",
			then:  "hotkeys:1: %delay x soon: soon is not a duration",
		},
		{
			name:  "a delay of no time",
			given: "%delay x 0s\n",
			then:  "hotkeys:1: %delay x 0s: 0s is not a duration",
		},
		{
			name:  "a delay of less than no time",
			given: "%delay x -1s\n",
			then:  "hotkeys:1: %delay x -1s: -1s is not a duration",
		},
		{
			name:  "a delay declared twice",
			given: "%delay x 1s\n%delay x 1s\n",
			then:  "hotkeys:2: %delay x is declared twice",
		},
		{
			name:  "a delay called any",
			given: "%delay any 1s\n",
			then:  "hotkeys:1: %delay any: any is every key, not a delay",
		},
		{
			name:  "a scope declared twice",
			given: "a:\n\tcommand-s { x }\na:\n",
			then:  "hotkeys:3: scope a is declared twice",
		},
		{
			name:  "a wildcard of two keys",
			given: "... command-s command-t { x }\n",
			then:  "hotkeys:1: ... takes one key",
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			_, err := Parse(strings.NewReader(c.given), "hotkeys", nil)

			if err == nil || err.Error() != c.then {
				t.Errorf("Parse = %v; want %s", err, c.then)
			}
		})
	}
}

func TestCanonOrdersTheModifiers(t *testing.T) {
	for _, c := range []struct {
		name  string
		given string
		then  string
	}{
		{name: "a bare key", given: "b", then: "b"},
		{name: "two modifiers out of order", given: "shift-command-b", then: "command-shift-b"},
		{name: "every modifier out of order", given: "option-control-shift-command-b", then: "command-shift-control-option-b"},
		{name: "a modifier on the minus key", given: "command--", then: "command--"},
		{name: "the minus key", given: "-", then: "-"},
		{name: "a modifier with nothing after it", given: "command-", then: "command-"},
		{name: "a modifier before a modifier with nothing after it", given: "shift-command-", then: "shift-command-"},
	} {
		t.Run(c.name, func(t *testing.T) {
			got, err := Canon(c.given)

			if err != nil || got != c.then {
				t.Errorf("Canon(%q) = %q, %v; want %q", c.given, got, err, c.then)
			}
		})
	}
}

func TestCanonRefusesAModifierTwice(t *testing.T) {
	for _, c := range []struct {
		name  string
		given string
		then  string
	}{
		{name: "next to each other", given: "command-command-s", then: "command twice"},
		{name: "apart", given: "shift-command-shift-s", then: "shift twice"},
	} {
		t.Run(c.name, func(t *testing.T) {
			_, err := Canon(c.given)

			if err == nil || err.Error() != c.then {
				t.Errorf("Canon(%q) = %v; want %s", c.given, err, c.then)
			}
		})
	}
}

func TestNameIsTheKeyUnderTheModifiers(t *testing.T) {
	for _, c := range []struct {
		name  string
		given string
		then  string
	}{
		{name: "a bare key", given: "b", then: "b"},
		{name: "a chord", given: "command-shift-b", then: "b"},
		{name: "the minus key", given: "control--", then: "-"},
		{name: "a modifier with nothing after it", given: "shift-command-", then: "command-"},
	} {
		t.Run(c.name, func(t *testing.T) {
			if got := Name(c.given); got != c.then {
				t.Errorf("Name(%q) = %q; want %q", c.given, got, c.then)
			}
		})
	}
}

func TestTheUSKeyboardRunsFromAToF16(t *testing.T) {
	for _, c := range []struct {
		name  string
		given int
		then  string
	}{
		{name: "the first letter", given: 0, then: "a"},
		{name: "the last letter", given: 25, then: "z"},
		{name: "the first digit", given: 26, then: "0"},
		{name: "the last digit", given: 35, then: "9"},
		{name: "the first mark", given: 36, then: "-"},
		{name: "the last mark", given: 46, then: "`"},
		{name: "the first named key", given: 47, then: "return"},
		{name: "the last named key", given: 62, then: "clear"},
		{name: "the first function key", given: 63, then: "f1"},
		{name: "the last function key", given: 78, then: "f16"},
	} {
		t.Run(c.name, func(t *testing.T) {
			if got := US[c.given]; got != c.then {
				t.Errorf("US[%d] = %q; want %q", c.given, got, c.then)
			}
		})
	}
	if len(US) != 79 {
		t.Errorf("the US keyboard has %d keys; want 79", len(US))
	}
}

func TestAKeyboardHasOnlyItsKeys(t *testing.T) {
	for _, c := range []struct {
		name  string
		given string
		then  bool
	}{
		{name: "a letter", given: "z", then: true},
		{name: "a function key", given: "f16", then: true},
		{name: "a function key past the last", given: "f17", then: false},
		{name: "a capital letter", given: "Z", then: false},
		{name: "a chord", given: "command-z", then: false},
	} {
		t.Run(c.name, func(t *testing.T) {
			if got := US.Has(c.given); got != c.then {
				t.Errorf("US.Has(%q) = %v; want %v", c.given, got, c.then)
			}
		})
	}
}

func TestTheAlphabetIsEveryChordOfTheKeyboard(t *testing.T) {
	given := Keyboard{"a", "b"}

	r, err := Parse(strings.NewReader("%delay settle 1s\nmain:\n\tb settle { x }\n"), "hotkeys", given)

	if err != nil {
		t.Fatal(err)
	}
	a := r.Table.Alpha
	got := make([]string, 0, a.Size())
	for s := range a.Size() {
		got = append(got, a.Name(seq.Sym(s)))
	}
	want := []string{
		"a", "b",
		"command-a", "command-b",
		"shift-a", "shift-b",
		"command-shift-a", "command-shift-b",
		"control-a", "control-b",
		"command-control-a", "command-control-b",
		"shift-control-a", "shift-control-b",
		"command-shift-control-a", "command-shift-control-b",
		"option-a", "option-b",
		"command-option-a", "command-option-b",
		"shift-option-a", "shift-option-b",
		"command-shift-option-a", "command-shift-option-b",
		"control-option-a", "control-option-b",
		"command-control-option-a", "command-control-option-b",
		"shift-control-option-a", "shift-control-option-b",
		"command-shift-control-option-a", "command-shift-control-option-b",
		"settle",
	}
	if !reflect.DeepEqual(got, want) || a.Keys() != 32 {
		t.Errorf("the alphabet is %q with %d keys; want %q with 32", got, a.Keys(), want)
	}
}

func TestAWildcardMayEndAPrefixOrLeaveAScope(t *testing.T) {
	r := parse(t, "%delay quiet 2s\n... escape { reset } -> main\nmain:\n\t-> edit when app == \"devdraw\"\nedit:\n\tcommand-k quiet { kill }\n\tcommand-k command-k { kill twice }\n\t... quiet { idle }\n")
	m := seqtest.Machine(t, r.Table, "")

	seqtest.Run(t, m, r.Table.Alpha, acme, []seqtest.Step{
		{Key: "command-k", Taken: true, Pending: true},
		{Key: "quiet", Taken: true, Fired: "kill"},
		{Key: "command-k", Taken: true, Pending: true},
		{Key: "escape", Taken: true, Fired: "reset"},
		{Key: "quiet", Taken: true, Fired: "idle"},
		{Key: "command-k", Taken: true, Pending: true},
		{Key: "command-k", Taken: true, Fired: "kill twice"},
	})
}

func TestAnyTakesWhatNoOtherRuleNames(t *testing.T) {
	r := parse(t, `%delay settle 1s

main:
	control-x control-s	{ put }
	control-x any		{ }
	command-m		-> mode

mode:
	j			{ down }
	escape			-> main
	... any			{ }
`)
	m := seqtest.Machine(t, r.Table, "")

	seqtest.Run(t, m, r.Table.Alpha, nil, []seqtest.Step{
		{Key: "control-x", Taken: true, Pending: true},
		{Key: "control-s", Taken: true, Fired: "put"},
		{Key: "control-x", Taken: true, Pending: true},
		{Key: "j", Taken: true},
		{Key: "command-s"},
		{Key: "command-m", Taken: true},
		{Key: "j", Taken: true, Fired: "down"},
		{Key: "k", Taken: true},
		{Key: "command-s", Taken: true},
		{Key: "settle"},
		{Key: "escape", Taken: true},
		{Key: "k"},
	})

	if waits := r.Table.Waiting(); len(waits) != 0 {
		t.Errorf("any reads the delays %q; it reads keys", waits)
	}
}

func TestPeekReadsTheNextKeyAgain(t *testing.T) {
	r := parse(t, `main:
	control-x control-s	{ put }
	control-x peek any	{ show }
	command-s		{ save }
`)
	m := seqtest.Machine(t, r.Table, "")

	seqtest.Run(t, m, r.Table.Alpha, nil, []seqtest.Step{
		{Key: "control-x", Taken: true, Pending: true},
		{Key: "control-s", Taken: true, Fired: "put"},
		{Key: "control-x", Taken: true, Pending: true},
		{Key: "command-s", Taken: true, Fired: "show,save"},
		{Key: "control-x", Taken: true, Pending: true},
		{Key: "j", Fired: "show"},
		{Key: "j"},
		{Key: "control-x", Taken: true, Pending: true},
		{Key: "control-x", Taken: true, Fired: "show", Pending: true},
	})

	if got := r.Grammar.Rules[1].Pattern; got != "control-x peek any" {
		t.Errorf("the pattern reads %q", got)
	}
}
