package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenseq(t *testing.T) {
	dir := t.TempDir()
	bad := filepath.Join(dir, "bad.tree")
	if err := os.WriteFile(bad, []byte("start:\n  0  do x  -> nowhere\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		args []string
		code int
		says string
	}{
		{nil, 2, "usage: genseq"},
		{[]string{"-d", "nope", "../../examples/dial.tree"}, 1, "no dialect called nope"},
		{[]string{bad}, 1, "-> nowhere, and there is no such scope"},
		{[]string{"-dot", dir, "../../examples/hotkeys", "../../examples/hotkeys"}, 1, "two tables are called hotkeys"},
		{[]string{"-v", "-dot", dir, "../../examples/dial.tree", "../../examples/hotkeys"}, 0, "dial, "},
	} {
		var stderr bytes.Buffer
		if code := run(c.args, &stderr); code != c.code || !strings.Contains(stderr.String(), c.says) {
			t.Errorf("genseq %v: %d %q; want %d and %q", c.args, code, stderr.String(), c.code, c.says)
		}
	}
	for _, name := range []string{"dial.dot", "hotkeys.dot"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Error(err)
		}
	}
}
