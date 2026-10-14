package seqtest_test

import (
	"strings"
	"testing"

	"github.com/cptaffe/automaton/seq/hotkey"
	"github.com/cptaffe/automaton/seq/seqtest"
)

func TestRunWalksAMachineThroughItsSteps(t *testing.T) {
	r, err := hotkey.Parse(strings.NewReader("main:\n\tcontrol-x control-s { save }\n\tcontrol-x any { }\n"), "h", nil)
	if err != nil {
		t.Fatal(err)
	}
	m := seqtest.Machine(t, r.Table, "")

	seqtest.Run(t, m, r.Table.Alpha, nil, []seqtest.Step{
		{Key: "control-x", Taken: true, Pending: true},
		{Key: "control-s", Taken: true, Fired: "save"},
		{Key: "j"},
		{Key: "not-a-key"},
	})
}
