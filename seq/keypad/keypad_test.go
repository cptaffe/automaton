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
	m, err := seq.NewMachine(tab, scope)
	if err != nil {
		t.Fatal(err)
	}
	var last *seq.Accept
	for _, name := range syms {
		sym := tab.Alpha.Of(name)
		if sym == 0 {
			t.Fatalf("no symbol %s", name)
		}
		if _, fired := m.Step(sym, seq.Env{"holding": "yes"}); len(fired) > 0 {
			last = fired[len(fired)-1]
		}
	}
	return last
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
	if got := strings.Join(tab.Waiting(), " "); got != "T abandon" {
		t.Errorf("waiting %v", got)
	}
	m, _ := seq.NewMachine(tab, "mailbox")
	if ok, _ := m.Step(tab.Alpha.Of("1"), nil); ok {
		t.Error("1 in the mailbox was taken while nothing is holding")
	}
}

func TestATreeHandsOff(t *testing.T) {
	g1, err := Parse(strings.NewReader("start:\n  0  do operator  => menu\n"), "dial.tree")
	if err != nil {
		t.Fatal(err)
	}
	g2, err := Parse(strings.NewReader("start:\n  1  do one\n"), "menu.tree")
	if err != nil {
		t.Fatal(err)
	}
	a := Alphabet(g1, g2)
	t1, _ := seq.Compile(g1, a)
	t2, _ := seq.Compile(g2, a)
	tab, err := seq.Link([]*seq.Table{t2, t1}, "all", Answered)
	if err != nil {
		t.Fatal(err)
	}
	if a := walk(t, tab, "", "0", Answered, "1"); a == nil || a.Action != "one" {
		t.Errorf("0, answered, 1 fired %+v; want the menu's one", a)
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

func TestCommentsLabelAndNote(t *testing.T) {
	g, err := Parse(strings.NewReader("; dial plan\n\nstart:\n  ; label: the operator\n  0  do operator\n  ; a remark\n  1  do one  when a and (b or not c)\n"), "p.tree")
	if err != nil {
		t.Fatal(err)
	}
	if g.Note != "dial plan" {
		t.Errorf("note %q", g.Note)
	}
	if g.Rules[0].Label != "the operator" || g.Rules[1].Label != "" {
		t.Errorf("labels %q %q", g.Rules[0].Label, g.Rules[1].Label)
	}
	if got := g.Rules[1].When.String(); got != "a and (b or not c)" {
		t.Errorf("when %q", got)
	}
}

func TestMoreErrors(t *testing.T) {
	for _, c := range []struct{ src, want string }{
		{"s:\n  1 do a 2\n", `"2" is a pattern symbol, and one cannot stand here`},
		{"s:\n  1 do a when\n", "cannot stand here"},
		{"s:\n  1 do a when x y\n", `"y" is a name, and one cannot stand here`},
		{"%bogus\n", "%bogus is not a declaration"},
		{"%delay T soon\n", "soon is not a duration"},
		{"%delay T 0s\n", "a wait of no time is not a wait"},
		{"%delay T 1s\n%delay T 2s\n", "it is already 1s"},
		{"%delay T 1s\ns:\n  %settles T\n  %settles U\n", "T already settles s"},
		{"s:\n  ... 1 2 do a\n", "is more than one symbol"},
		{"s:\n  ... 1 do Put\n", "not lower case"},
	} {
		_, err := Parse(strings.NewReader(c.src), "t.tree")
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%q: got %v, want %q", c.src, err, c.want)
		}
	}
}
