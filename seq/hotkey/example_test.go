package hotkey

import (
	"os"
	"testing"

	"github.com/cptaffe/automaton/internal/seqtest"
)

func TestTheExampleRuns(t *testing.T) {
	f, err := os.Open("../../examples/hotkeys")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	r, err := Parse(f, "hotkeys", nil)
	if err != nil {
		t.Fatal(err)
	}
	m := seqtest.Machine(t, r.Table, "")

	seqtest.Run(t, m, r.Table.Alpha, acme, []seqtest.Step{
		{Key: "control-x", Taken: true, Pending: true},
		{Key: "control-s", Taken: true, Fired: "ctl put"},
		{Key: "control-x", Taken: true, Pending: true},
		{Key: "j", Taken: true},
		{Key: "control-x", Taken: true, Pending: true},
		{Key: "settle", Taken: true},
		{Key: "command-w", Taken: true, Fired: "ctl del"},
	})
}
