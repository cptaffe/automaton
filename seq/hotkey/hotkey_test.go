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

acme: when app == "devdraw" || app ~ "^9term$"
	%settles settle
	command-s            { put }                 # save
	shift-command-b      { Lrefs }               # references
	control-x control-s  { put }
	control-x            { echo '}' {nested} }   # look
	command--            { Shrink }
	command-m            -> mode

mode:
	j                    { down }
	escape               { exit } -> acme
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

func TestMatch(t *testing.T) {
	r := parse(t, keytab)
	acme := seq.Env{"app": "devdraw"}
	m := r.Matcher()
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
		{"command-s", acme, true, "put", nil},
	} {
		consume, fired, waits := m.Key(c.key, c.env)
		if consume != c.consume || actions(fired) != c.fired || !reflect.DeepEqual(waits, c.waits) {
			t.Errorf("%s in %v: got %v %q %v, want %v %q %v", c.key, c.env["app"], consume, actions(fired), waits, c.consume, c.fired, c.waits)
		}
	}
}

func TestSettleOnTimeout(t *testing.T) {
	r := parse(t, keytab)
	m := r.Matcher()
	m.Key("control-x", seq.Env{"app": "devdraw"})
	fired, waits := m.Event("settle", nil)
	if actions(fired) != "echo '}' {nested}" || waits != nil {
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
		{"a: when app ~ \"(\"\n", "missing closing )"},
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
