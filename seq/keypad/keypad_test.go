package keypad

import (
	"strings"
	"testing"

	"github.com/cptaffe/automaton/seq"
)

const plan = `%delay T 5s
%delay abandon 16s

start:
  %settles T
  *98           do voicemail  -> mailbox
  0             do operator
  NXX-XXXX      do line
  960-XXX-XXXX  do line-ten
  ... abandon   do dial-again

mailbox:
  1             do play       when holding
  #             do stop       -> start
`

func compile(t *testing.T, src string) *seq.Table {
	t.Helper()
	g, err := Parse(strings.NewReader(src), "plan.tree")
	if err != nil {
		t.Fatal(err)
	}
	tab, err := seq.Compile(g, Alphabet(g))
	if err != nil {
		t.Fatal(err)
	}
	return tab
}

func walk(t *testing.T, tab *seq.Table, scope string, syms ...string) *seq.Accept {
	t.Helper()
	s := tab.Start[scope]
	for _, name := range syms {
		sym, ok := tab.Alpha.Of(name)
		if !ok {
			t.Fatalf("no symbol %s", name)
		}
		if n := tab.Next[s][sym]; n != seq.Dead {
			s = n
			continue
		}
		return tab.Any[s][sym]
	}
	return tab.Accept[s]
}

func TestPlan(t *testing.T) {
	tab := compile(t, plan)
	if len(tab.Conflicts) > 0 {
		t.Fatal(tab.Conflicts)
	}
	for _, c := range []struct {
		scope  string
		keys   string
		action string
	}{
		{"start", "* 9 8", "voicemail"},
		{"start", "0", "operator"},
		{"start", "4 8 2 1 2 3 4", "line"},
		{"start", "9 6 0 1 2 3 4 T", "line"},
		{"start", "9 6 0 4 8 2 1 2 3 4", "line-ten"},
		{"start", "4 abandon", "dial-again"},
		{"mailbox", "1", "play"},
	} {
		a := walk(t, tab, c.scope, strings.Fields(c.keys)...)
		if a == nil || a.Action != c.action {
			t.Errorf("%s %s: got %+v, want %s", c.scope, c.keys, a, c.action)
		}
	}
	if a := walk(t, tab, "start", "*", "9", "8"); a.Goto != "mailbox" {
		t.Errorf("*98 goes to %q", a.Goto)
	}
	if a := walk(t, tab, "mailbox", "1"); a == nil || a.When == nil || a.When.String() != "holding" {
		t.Errorf("when %v", a.When)
	}
	if a := walk(t, tab, "start", "4", "8", "2", "1", "2", "3", "4"); a == nil || a.Operand == nil || *a.Operand != (seq.Slice{From: 0, To: 0}) {
		t.Errorf("operand %+v", a.Operand)
	}
	if got := tab.Waiting(); len(got) != 1 || got[0] != "T" {
		t.Errorf("waiting %v", got)
	}
}

func TestConflicts(t *testing.T) {
	for _, c := range []struct{ src, want string }{
		{"s:\n  1 do a\n  1 do b\n", "two rules, one key sequence"},
		{"s:\n  1 do a\n  12 do b\n", "has no %settles"},
	} {
		tab := compile(t, c.src)
		if len(tab.Conflicts) != 1 || !strings.Contains(tab.Conflicts[0], c.want) {
			t.Errorf("%q: %v", c.src, tab.Conflicts)
		}
	}
}

func TestErrors(t *testing.T) {
	for _, c := range []struct{ src, want string }{
		{"1 do a\n", "above every scope"},
		{"s:\n  1\n", "nowhere to go"},
		{"s:\n  1 do Put\n", "not lower case"},
		{"s:\n  1 do a -> t\n", "no such scope"},
		{"s:\n  1X+ do a\n", "ends in a repeat"},
		{"s:\n  %delay T 1s\n", "t.tree:2: %delay T 1s belongs above every scope"},
		{"\n%settles T\n", "t.tree:2: %settles T is above every scope"},
		{"s:\n  1 do a\n\n  2 do Put\n", "t.tree:4: the action"},
	} {
		_, err := Parse(strings.NewReader(c.src), "t.tree")
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%q: got %v, want %q", c.src, err, c.want)
		}
	}
}
