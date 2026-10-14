package seqtest_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/cptaffe/automaton/seq"
	"github.com/cptaffe/automaton/seq/hotkey"
	"github.com/cptaffe/automaton/seq/seqtest"
)

const rules = `main:
	-> acme when app == "acme"
	control-x control-s { save }
	control-x any { }

acme:
	command-s { put }
`

func table(t *testing.T) *seq.Table {
	t.Helper()
	r, err := hotkey.Parse(strings.NewReader(rules), "h", nil)
	if err != nil {
		t.Fatal(err)
	}
	return r.Table
}

type recorder struct {
	testing.TB
	failed []string
}

func (r *recorder) Helper() {}

func (r *recorder) Fatal(a ...any) { r.failed = append(r.failed, fmt.Sprint(a...)) }

func (r *recorder) Fatalf(f string, a ...any) { r.failed = append(r.failed, fmt.Sprintf(f, a...)) }

func TestRunWalksAMachineThroughItsSteps(t *testing.T) {
	tab := table(t)
	m := seqtest.Machine(t, tab, "")

	seqtest.Run(t, m, tab.Alpha, nil, []seqtest.Step{
		{Key: "control-x", Taken: true, Pending: true},
		{Key: "control-s", Taken: true, Fired: "save"},
		{Key: "control-x", Taken: true, Pending: true},
		{Key: "j", Taken: true},
		{Key: "control-x", Taken: true, Pending: true},
		{Key: "not-a-key"},
		{Key: "j"},
	})
}

func TestAStepsEnvTakesThePlaceOfTheRunsEnv(t *testing.T) {
	tab := table(t)
	m := seqtest.Machine(t, tab, "")

	seqtest.Run(t, m, tab.Alpha, seq.Env{"app": "acme"}, []seqtest.Step{
		{Key: "command-s", Taken: true, Fired: "put"},
		{Key: "command-s", Env: seq.Env{"app": "Safari"}},
	})
}

func TestRunStopsAtTheFirstStepThatDiffers(t *testing.T) {
	for _, c := range []struct {
		name  string
		given []seqtest.Step
		then  []string
	}{
		{
			name:  "a key taken that should be left",
			given: []seqtest.Step{{Key: "control-x"}, {Key: "j"}},
			then:  []string{`step 1, control-x: taken true, fired "", pending true; want taken false, fired "", pending false`},
		},
		{
			name:  "an action that did not fire",
			given: []seqtest.Step{{Key: "control-x", Taken: true, Pending: true}, {Key: "control-s", Taken: true, Fired: "put"}},
			then:  []string{`step 2, control-s: taken true, fired "save", pending false; want taken true, fired "put", pending false`},
		},
		{
			name:  "a machine left waiting",
			given: []seqtest.Step{{Key: "control-x", Taken: true}},
			then:  []string{`step 1, control-x: taken true, fired "", pending true; want taken true, fired "", pending false`},
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			tab := table(t)
			r := &recorder{TB: t}

			seqtest.Run(r, seqtest.Machine(t, tab, ""), tab.Alpha, nil, c.given)

			if !reflect.DeepEqual(r.failed, c.then) {
				t.Errorf("Run failed with %q; want %q", r.failed, c.then)
			}
		})
	}
}

func TestPrefixTakesEachKeyAndWaits(t *testing.T) {
	given := "4  8 2"

	when := seqtest.Prefix(given)

	then := []seqtest.Step{
		{Key: "4", Taken: true, Pending: true},
		{Key: "8", Taken: true, Pending: true},
		{Key: "2", Taken: true, Pending: true},
	}
	if !reflect.DeepEqual(when, then) {
		t.Errorf("Prefix(%q) = %+v; want %+v", given, when, then)
	}
}

func TestActionsJoinsWhatFired(t *testing.T) {
	for _, c := range []struct {
		name  string
		given []*seq.Accept
		then  string
	}{
		{name: "nothing", given: nil, then: ""},
		{name: "one action", given: []*seq.Accept{{Action: "save"}}, then: "save"},
		{name: "two actions", given: []*seq.Accept{{Action: "show"}, {Action: "save"}}, then: "show,save"},
	} {
		t.Run(c.name, func(t *testing.T) {
			if got := seqtest.Actions(c.given); got != c.then {
				t.Errorf("Actions = %q; want %q", got, c.then)
			}
		})
	}
}

func TestMachineFailsOnAScopeThatIsNotThere(t *testing.T) {
	tab := table(t)
	r := &recorder{TB: t}

	seqtest.Machine(r, tab, "nowhere")

	if want := []string{`h has no scope "nowhere"`}; !reflect.DeepEqual(r.failed, want) {
		t.Errorf("Machine failed with %q; want %q", r.failed, want)
	}
}
