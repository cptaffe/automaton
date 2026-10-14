package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cptaffe/automaton/dot/dottest"
)

type drawn struct {
	name  string
	note  string
	nodes int
	edges int
}

type result struct {
	code int
	says string
}

func TestGenseqExitsWithACodeAndSaysWhy(t *testing.T) {
	dir := t.TempDir()
	bad := filepath.Join(dir, "bad.tree")
	if err := os.WriteFile(bad, []byte("start:\n  0  do x  -> nowhere\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	twice := filepath.Join(dir, "twice.tree")
	if err := os.WriteFile(twice, []byte("start:\n  1 do a\n  1 do b\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		name  string
		given []string
		when  []string
		then  result
	}{
		{"no file is a usage error", nil, nil, result{2, "usage: genseq [-v] [-d dialect] [-dot dir] file...\n"}},
		{"an unknown flag is a usage error", []string{"../../examples/dial.tree"}, []string{"-x"}, result{2, "flag provided but not defined: -x"}},
		{"a missing file is an error", []string{"missing.tree"}, nil, result{1, "genseq: open missing.tree: no such file or directory\n"}},
		{"an unknown dialect is an error", []string{"../../examples/dial.tree"}, []string{"-d", "nope"}, result{1, "genseq: no dialect called nope\n"}},
		{"a hotkey file read as keypad is an error", []string{"../../examples/hotkeys"}, []string{"-d", "keypad"}, result{1, "no rule reads '%'"}},
		{"a goto to no scope is an error", []string{bad}, nil, result{1, "-> nowhere, and there is no such scope"}},
		{"a conflict is an error", []string{twice}, nil, result{1, `"1" and "1" both finish here -- two rules, one key sequence`}},
		{"two drawings of one name are an error", []string{"../../examples/hotkeys", "../../examples/hotkeys"}, []string{"-dot", dir}, result{1, "genseq: two tables are called hotkeys, and their drawings would share hotkeys.dot\n"}},
		{"two tables of one name are fine without drawings", []string{"../../examples/hotkeys", "../../examples/hotkeys"}, nil, result{0, ""}},
		{"verbose describes each table, keypad last", []string{"../../examples/dial.tree", "../../examples/hotkeys"}, []string{"-v"}, result{0, "genseq: hotkeys, 8 states, waits for settle\ngenseq: dial, 38 states, waits for T abandon\n"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			var stderr bytes.Buffer
			code := run(append(c.when, c.given...), &stderr)
			got := result{code, stderr.String()}
			if got.code != c.then.code || !strings.Contains(got.says, c.then.says) || c.then.says == "" && got.says != "" {
				t.Errorf("got %d %q; want %d %q", got.code, got.says, c.then.code, c.then.says)
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
