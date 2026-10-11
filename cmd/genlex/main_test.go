package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenlex(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "scan", "scan.gen.go")
	for _, c := range []struct {
		args []string
		code int
		says string
	}{
		{nil, 2, "usage: genlex"},
		{[]string{"-nope"}, 2, "flag provided but not defined"},
		{[]string{"-go", out, "a.lex", "b.lex"}, 1, "takes one file"},
		{[]string{"-go", out, "../../examples/comments.lex"}, 1, "-go needs -package"},
		{[]string{"missing.lex"}, 1, "no such file"},
		{[]string{"-v", "-dot", dir, "-go", out, "-package", "scan", "../../examples/comments.lex"}, 0, "start, 4 rules"},
	} {
		var stderr bytes.Buffer
		if code := run(c.args, &stderr); code != c.code || !strings.Contains(stderr.String(), c.says) {
			t.Errorf("genlex %v: %d %q; want %d and %q", c.args, code, stderr.String(), c.code, c.says)
		}
	}
	for _, name := range []string{out, filepath.Join(dir, "comments.dot"), filepath.Join(dir, "comments-shape.dot")} {
		if _, err := os.Stat(name); err != nil {
			t.Error(err)
		}
	}
}
