package hotkey

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/cptaffe/automaton/seq"
)

const keytab = `# Edit ,>9p write acme-hotkey/rules
%{
fn put { echo put | 9p write acme/$winid/ctl }
%}
%delay settle 1s

main:
	-> acme when app == "devdraw" || app ~ "^9term$"

acme:
	%settles settle
	command-s            { put }                 # save
	shift-command-b      { Lrefs }               # references
	control-x control-s  { put }
	control-x            { echo '}' {nested} }   # look
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

func actions(as []*seq.Accept) string {
	var s []string
	for _, a := range as {
		s = append(s, a.Action)
	}
	return strings.Join(s, ",")
}

func TestParse(t *testing.T) {
	r := parse(t, keytab)
	if got := r.Prelude; got != "fn put { echo put | 9p write acme/$winid/ctl }" {
		t.Errorf("prelude %q", got)
	}
	labels := map[string]string{}
	for _, rule := range r.Grammar.Rules {
		labels[rule.Pattern] = rule.Label
	}
	if labels["command-s"] != "save" || labels["control-x"] != "look" {
		t.Errorf("labels %v", labels)
	}
	var look string
	for _, rule := range r.Grammar.Rules {
		if rule.Pattern == "control-x" {
			look = rule.Action
		}
	}
	if look != "echo '}' {nested}" {
		t.Errorf("a block with a quoted brace and a nested block reads %q", look)
	}
}

func machine(t *testing.T, r *Rules) *seq.Machine {
	t.Helper()
	m, err := seq.NewMachine(r.Table, "")
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func TestMatch(t *testing.T) {
	r := parse(t, keytab)
	acme := seq.Env{"app": "devdraw"}
	m := machine(t, r)
	for _, c := range []struct {
		key     string
		env     seq.Env
		consume bool
		fired   string
		waits   []string
	}{
		{"command-s", acme, true, "put", nil},
		{"command-shift-b", acme, true, "Lrefs", nil},
		{"command--", acme, true, "Shrink", nil},
		{"command-s", seq.Env{"app": "9term"}, true, "put", nil},
		{"command-s", seq.Env{"app": "Safari"}, false, "", nil},
		{"space", acme, false, "", nil},
		{"control-x", acme, true, "", []string{"settle"}},
		{"control-s", acme, true, "put", nil},
		{"control-x", acme, true, "", []string{"settle"}},
		{"space", acme, false, "echo '}' {nested}", nil},
		{"control-x", acme, true, "", []string{"settle"}},
		{"command-s", acme, true, "echo '}' {nested},put", nil},
		{"command-m", acme, true, "", nil},
		{"j", seq.Env{"app": "Safari"}, true, "down", nil},
		{"command-s", acme, false, "", nil},
		{"escape", acme, true, "exit", nil},
		{"j", acme, false, "", nil},
		{"command-s", seq.Env{"app": "Safari"}, false, "", nil},
		{"command-s", acme, true, "put", nil},
	} {
		consume, fired := m.Step(r.Table.Alpha.Of(c.key), c.env)
		waits := m.Waits()
		if consume != c.consume || actions(fired) != c.fired || !reflect.DeepEqual(waits, c.waits) {
			t.Errorf("%s in %v: got %v %q %v, want %v %q %v", c.key, c.env["app"], consume, actions(fired), waits, c.consume, c.fired, c.waits)
		}
	}
}

func TestSettleOnTimeout(t *testing.T) {
	r := parse(t, keytab)
	m := machine(t, r)
	m.Step(r.Table.Alpha.Of("control-x"), seq.Env{"app": "devdraw"})
	_, fired := m.Step(r.Table.Alpha.Of("settle"), seq.Env{"app": "devdraw"})
	if waits := m.Waits(); actions(fired) != "echo '}' {nested}" || waits != nil {
		t.Errorf("settle fired %q, waits %v", actions(fired), waits)
	}
}

func TestErrors(t *testing.T) {
	known := func(name string) error {
		if name == "zz" {
			return errors.New("no such key")
		}
		return nil
	}
	for _, c := range []struct{ src, want string }{
		{"a:\n\tcommand-s { x }\n\tcommand-s { y }\n", "two rules, one key sequence"},
		{"a:\n\tcontrol-x { x }\n\tcontrol-x control-s { y }\n", "no %settles"},
		{"a:\n\tcommand-zz { x }\n", "hotkeys:2: command-zz: no such key"},
		{"a:\n\tcommand-s { x\n", "never closed"},
		{"command-s { x }\n", "above every scope"},
		{"a:\n\tcommand-s\n", "nowhere to go"},
		{"a:\n\tcommand-command-s { x }\n", "command twice"},
		{"a:\n\t-> a when app ~ \"(\"\n", "missing closing )"},
		{"a: when app == \"x\"\n", "when cannot stand here"},
		{"-> a\na:\n\tcommand-s { x }\n", "above every scope"},
		{"a:\n\t-> b\nb:\n\t-> a\n", "goes round in a loop"},
		{"a:\n\tcommand-s -> b\n", "no such scope"},
		{"%{\nfn x\n", "hotkeys:"},
	} {
		_, err := Parse(strings.NewReader(c.src), "hotkeys", known)
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%q: got %v, want %q", c.src, err, c.want)
		}
	}
}

func TestCanon(t *testing.T) {
	for in, want := range map[string]string{
		"shift-command-b":  "command-shift-b",
		"command--":        "command--",
		"option-control-x": "control-option-x",
		"-":                "-",
	} {
		if got, err := Canon(in); err != nil || got != want {
			t.Errorf("Canon(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
}

func TestAWildcardAndDelays(t *testing.T) {
	r := parse(t, "%delay quiet 2s\n... escape { reset } -> main\nmain:\n\t-> edit when app == \"devdraw\"\nedit:\n\t%settles quiet\n\tcommand-k { kill }\n\tcommand-k command-k { kill twice }\n\t... quiet { idle }\n")
	m := machine(t, r)
	env := seq.Env{"app": "devdraw"}
	for _, c := range []struct {
		key   string
		fired string
	}{
		{"command-k", ""},
		{"quiet", "kill"},
		{"escape", "reset"},
		{"quiet", "idle"},
	} {
		if _, fired := m.Step(r.Table.Alpha.Of(c.key), env); actions(fired) != c.fired {
			t.Errorf("%s fired %q; want %q", c.key, actions(fired), c.fired)
		}
	}
}

func TestDeclarationErrors(t *testing.T) {
	for _, c := range []struct{ src, want string }{
		{"a:\n\t%delay x 1s\n", "belongs above every scope"},
		{"%delay x soon\n", "soon is not a duration"},
		{"%delay x 1s\n%delay x 1s\n", "declared twice"},
		{"%settles x\n", "above every scope"},
		{"%delay x 1s\n%delay y 1s\na:\n\t%settles x\n\t%settles y\n", "x already settles a"},
		{"a:\n\tcommand-s { x }\na:\n", "scope a is declared twice"},
		{"... command-s command-t { x }\n", "... takes one key"},
		{"a:\n\tcommand-s { x }\n\t-> nowhere\n", "no such scope"},
	} {
		_, err := Parse(strings.NewReader(c.src), "hotkeys", nil)
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%q: got %v, want %q", c.src, err, c.want)
		}
	}
}
