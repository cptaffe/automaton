package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestLrdot(t *testing.T) {
	dir := t.TempDir()
	report := filepath.Join(dir, "y.output")
	cmd := exec.Command("go", "tool", "goyacc", "-l", "-p", "yy", "-o", os.DevNull, "-v", report, "gram.y")
	cmd.Dir = "../../seq/keypad"
	if b, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("goyacc: %v\n%s", err, b)
	}
	out := filepath.Join(dir, "gram.dot")
	for _, c := range []struct {
		args []string
		code int
		says string
	}{
		{nil, 2, "usage: lrdot"},
		{[]string{"missing", out}, 1, "no such file"},
		{[]string{"-name", "keypad", report, out}, 0, ""},
	} {
		var stderr bytes.Buffer
		if code := run(c.args, &stderr); code != c.code || !strings.Contains(stderr.String(), c.says) {
			t.Errorf("lrdot %v: %d %q; want %d and %q", c.args, code, stderr.String(), c.code, c.says)
		}
	}
	for _, name := range []string{out, filepath.Join(dir, "gram-tokens.dot")} {
		b, err := os.ReadFile(name)
		if err != nil || !bytes.Contains(b, []byte("keypad")) {
			t.Errorf("%s: %v", name, err)
		}
	}
}
