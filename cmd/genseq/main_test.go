package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/cptaffe/automaton/internal/dottest"
)

const usage = `usage: genseq [-v] [-d dialect] [-dot dir] file...
  -d string
    	the dialect, keypad or hotkey; by default .tree is keypad and anything else hotkey
  -dot string
    	write Graphviz drawings into this directory
  -v	describe each table
`

var files = map[string]string{
	"one.tree":   "start:\n  1  do one  -> two\n",
	"two.tree":   "%delay T 1s\nstart:\n  2 T  do two\n",
	"bad.tree":   "start:\n  0  do x  -> nowhere\n",
	"both.tree":  "start:\n  1  do one\n  1  do two\n",
	"keys":       "main:\n\tcommand-s { save }\n",
	"keys.tree":  "main:\n\tcommand-s { save }\n",
	"other/keys": "main:\n\tcommand-w { close }\n",
}

type drawn struct {
	name  string
	note  string
	nodes int
	edges int
}

func TestGenseqReportsWhatItRead(t *testing.T) {
	for _, c := range []struct {
		name  string
		given []string
		then  int
		says  string
	}{
		{name: "no files", given: nil, then: 2, says: usage},
		{name: "a flag it does not know", given: []string{"-x", "keys"}, then: 2, says: "flag provided but not defined: -x\n" + usage},
		{name: "a dialect it does not know", given: []string{"-d", "nope", "two.tree"}, then: 1, says: "genseq: no dialect called nope\n"},
		{name: "a file that is not there", given: []string{"nofile"}, then: 1, says: "genseq: open nofile: no such file or directory\n"},
		{name: "a tree that does not compile", given: []string{"bad.tree"}, then: 1, says: "genseq: bad:2: -> nowhere, and there is no such scope\n"},
		{name: "a tree with a conflict", given: []string{"both.tree"}, then: 1, says: "genseq: both:3: \"1\" and \"1\" both finish here -- two rules, one key sequence\n"},
		{name: "a tree read as hotkeys", given: []string{"-d", "hotkey", "bad.tree"}, then: 1, says: "genseq: bad:2: do: no such key\n"},
		{name: "a dialect chosen by flag", given: []string{"-v", "-d", "hotkey", "keys.tree"}, then: 0, says: "genseq: keys, 2 states, waits for \n"},
		{name: "quietly", given: []string{"two.tree", "keys"}, then: 0, says: ""},
		{
			name:  "hotkeys first, then the trees linked as one",
			given: []string{"-v", "one.tree", "keys", "two.tree"},
			then:  0,
			says:  "genseq: keys, 2 states, waits for \ngenseq: trees, 5 states, waits for T\n",
		},
		{name: "two tables of one name drawn", given: []string{"-dot", "out", "keys", "other/keys"}, then: 1, says: "genseq: two tables are called keys, and their drawings would share keys.dot\n"},
		{name: "two tables of one name not drawn", given: []string{"keys", "other/keys"}, then: 0, says: ""},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Chdir(t.TempDir())
			for name, src := range files {
				if err := os.MkdirAll(filepath.Dir(name), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(name, []byte(src), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			var stderr bytes.Buffer

			code := run(c.given, &stderr)

			if code != c.then || stderr.String() != c.says {
				t.Errorf("genseq %q: %d and\n%s\nwant %d and\n%s", c.given, code, stderr.String(), c.then, c.says)
			}
		})
	}
}

func TestGenseqDrawsEachTable(t *testing.T) {
	given := t.TempDir()

	code := run([]string{"-dot", filepath.Join(given, "out"), "../../examples/dial.tree", "../../examples/hotkeys"}, os.Stderr)

	if code != 0 {
		t.Fatalf("exit %d; want 0", code)
	}
	for _, c := range []struct {
		name  string
		given string
		when  string
		then  drawn
	}{
		{"the dial plan is drawn", given, "dial.dot", drawn{"dial", `dial -- 38 states, 11 classes\l`, 38, 47}},
		{"the hotkeys are drawn", given, "hotkeys.dot", drawn{"hotkeys", `hotkeys -- 8 states, 7 classes\l`, 8, 7}},
	} {
		t.Run(c.name, func(t *testing.T) {
			b, err := os.ReadFile(filepath.Join(c.given, "out", c.when))
			if err != nil {
				t.Fatal(err)
			}
			d := dottest.Parse(t, b)
			if got := (drawn{d.Name, d.Note, len(d.Nodes), len(d.Edges)}); got != c.then {
				t.Errorf("got %+v; want %+v", got, c.then)
			}
		})
	}
}

func TestGenseqDrawsNothingWithoutADirectory(t *testing.T) {
	plan, err := filepath.Abs("../../examples/dial.tree")
	if err != nil {
		t.Fatal(err)
	}
	given := t.TempDir()
	t.Chdir(given)

	code := run([]string{plan}, os.Stderr)

	files, err := os.ReadDir(given)
	if code != 0 || err != nil || len(files) != 0 {
		t.Errorf("exit %d, %v and %d files; want 0, no error and no files", code, err, len(files))
	}
}
