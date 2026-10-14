package main

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cptaffe/automaton/internal/dottest"
)

func report(t *testing.T) string {
	t.Helper()
	out := filepath.Join(t.TempDir(), "y.output")
	cmd := exec.Command("go", "tool", "goyacc", "-l", "-p", "yy", "-o", os.DevNull, "-v", out, "gram.y")
	cmd.Dir = "../../seq/keypad"
	if b, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("goyacc: %v\n%s", err, b)
	}
	return out
}

type result struct {
	code int
	says string
}

type drawn struct {
	name  string
	note  string
	nodes int
	edges int
}

func parse(t *testing.T, path string) drawn {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	d := dottest.Parse(t, b)
	return drawn{d.Name, d.Note, len(d.Nodes), len(d.Edges)}
}

func TestLrdotExitsWithACodeAndSaysWhy(t *testing.T) {
	in := report(t)
	dir := t.TempDir()
	empty := filepath.Join(dir, "empty.output")
	if err := os.WriteFile(empty, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		name  string
		given []string
		when  []string
		then  result
	}{
		{"no file is a usage error", nil, nil, result{2, "usage: lrdot [-name n] [-shape-only] y.output out.dot\n"}},
		{"one file is a usage error", []string{in}, nil, result{2, "usage: lrdot [-name n] [-shape-only] y.output out.dot\n"}},
		{"three files are a usage error", []string{in, "a.dot", "b.dot"}, nil, result{2, "usage: lrdot [-name n] [-shape-only] y.output out.dot\n"}},
		{"an unknown flag is a usage error", []string{in, "a.dot"}, []string{"-x"}, result{2, "flag provided but not defined: -x"}},
		{"a missing report is an error", []string{"missing", filepath.Join(dir, "a.dot")}, nil, result{1, "lrdot: open missing: no such file or directory\n"}},
		{"a drawing that cannot be written is an error", []string{in, filepath.Join(dir, "no", "a.dot")}, nil, result{1, "no such file or directory\n"}},
		{"a report with no rules is an error", []string{empty, filepath.Join(dir, "a.dot")}, nil, result{1, "lrdot: " + empty + ": no grammar rules"}},
		{"a drawing that can be written says nothing", []string{in, filepath.Join(dir, "a.dot")}, nil, result{0, ""}},
	} {
		t.Run(c.name, func(t *testing.T) {
			var stderr bytes.Buffer
			code := run(append(c.when, c.given...), &stderr)
			if code != c.then.code || !strings.Contains(stderr.String(), c.then.says) || c.then.says == "" && stderr.Len() > 0 {
				t.Errorf("got %d %q; want %d %q", code, stderr.String(), c.then.code, c.then.says)
			}
		})
	}
}

func TestLrdotDrawsTheShapeAndTheTokens(t *testing.T) {
	given := t.TempDir()

	code := run([]string{"-name", "keypad", report(t), filepath.Join(given, "gram.dot")}, os.Stderr)

	if code != 0 {
		t.Fatalf("exit %d; want 0", code)
	}
	for _, c := range []struct {
		name  string
		given string
		when  string
		then  drawn
	}{
		{"the shape is drawn", given, "gram.dot", drawn{"keypad", "", 18, 30}},
		{
			"the tokens are drawn beside it", given, "gram-tokens.dot",
			drawn{"keypad", `keypad over tokens -- 20 states, 19 tokens; atom, cat, diff, expr, factor, pattern, piece, term self-embed, so this reads more than the grammar does\l`, 20, 52},
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			if got := parse(t, filepath.Join(c.given, c.when)); got != c.then {
				t.Errorf("got %+v; want %+v", got, c.then)
			}
		})
	}
}

func TestLrdotNamesTheShapeAfterItsFileAndCanLeaveOutTheTokens(t *testing.T) {
	given := t.TempDir()

	code := run([]string{"-shape-only", report(t), filepath.Join(given, "other.dot")}, os.Stderr)

	if code != 0 {
		t.Fatalf("exit %d; want 0", code)
	}
	if got, want := parse(t, filepath.Join(given, "other.dot")), (drawn{"other", "", 18, 30}); got != want {
		t.Errorf("got %+v; want %+v", got, want)
	}
	if _, err := os.Stat(filepath.Join(given, "other-tokens.dot")); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("the tokens were drawn too: %v", err)
	}
}
