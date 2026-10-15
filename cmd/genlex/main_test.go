package main

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"
)

func TestGenlexRefusesWhatItCannotDo(t *testing.T) {
	out := filepath.Join(t.TempDir(), "scan.gen.go")
	for _, c := range []struct {
		name  string
		given []string
		then  string
		code  int
	}{
		{"no file", nil, "usage: genlex", 2},
		{"an unknown flag", []string{"-nope"}, "flag provided but not defined: -nope", 2},
		{"one table from two files", []string{"-go", out, "a.lex", "b.lex"}, "genlex: -go writes one table, so it takes one file\n", 1},
		{"a table with no package", []string{"-go", out, "../../examples/comments.lex"}, "genlex: -go needs -package\n", 1},
		{"a missing file", []string{"missing.lex"}, "genlex: open missing.lex: no such file or directory\n", 1},
		{"a package Go cannot name", []string{"-go", out, "-package", "a-b", "../../examples/comments.lex"}, "genlex: ../../examples/comments.lex: \"a-b\" cannot name a package\n", 1},
	} {
		t.Run(c.name, func(t *testing.T) {
			var stderr bytes.Buffer

			code := run(c.given, &stderr)

			if code != c.code || !strings.HasPrefix(stderr.String(), c.then) {
				t.Errorf("exit %d, %q; want %d, beginning %q", code, stderr.String(), c.code, c.then)
			}
		})
	}
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		t.Errorf("a refused run wrote %s: %v", out, err)
	}
}

func TestGenlexWritesTheTableAndTheDrawings(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "scan", "scan.gen.go")
	var stderr bytes.Buffer

	code := run([]string{"-v", "-dot", dir, "-go", out, "-package", "scan", "../../examples/comments.lex"}, &stderr)

	if want := "genlex: start, 4 rules, 6 states, 6 classes\ngenlex: comment, 3 rules, 6 states, 6 classes\n"; code != 0 || stderr.String() != want {
		t.Fatalf("exit %d, %q; want 0, %q", code, stderr.String(), want)
	}
	file, err := parser.ParseFile(token.NewFileSet(), out, nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := names(file), []string{"TokClose", "TokNewline", "TokOpen", "TokWord", "Spec"}; file.Name.Name != "scan" || !reflect.DeepEqual(got, want) {
		t.Errorf("package %s declares %q; want package scan declaring %q", file.Name.Name, got, want)
	}
	if got, want := lines(t, filepath.Join(dir, "comments.dot"), `(?m)^\s+("m\d+_s\d+" -> "m\d+_s\d+")`), []string{
		`"m0_s1" -> "m0_s2"`, `"m0_s1" -> "m0_s3"`, `"m0_s1" -> "m0_s4"`, `"m0_s1" -> "m0_s5"`,
		`"m0_s2" -> "m0_s2"`, `"m0_s4" -> "m0_s6"`, `"m0_s5" -> "m0_s5"`, `"m1_s1" -> "m1_s2"`,
		`"m1_s1" -> "m1_s3"`, `"m1_s1" -> "m1_s4"`, `"m1_s3" -> "m1_s5"`, `"m1_s4" -> "m1_s6"`,
		`"m0_s6" -> "m1_s1"`,
	}; !reflect.DeepEqual(got, want) {
		t.Errorf("the drawing's edges are %q; want %q", got, want)
	}
	if got, want := lines(t, filepath.Join(dir, "comments-shape.dot"), `(?m)^\s+("\w+" -> "\w+")`), []string{
		`"start" -> "comment"`, `"comment" -> "comment"`,
	}; !reflect.DeepEqual(got, want) {
		t.Errorf("the shape's edges are %q; want %q", got, want)
	}
}

func TestGenlexWarnsOfAShadowedName(t *testing.T) {
	given := filepath.Join(t.TempDir(), "w.lex")
	source(t, given, "digit = [0-9]\n\nstart {\n  digit  emit d\n}\n")
	var stderr bytes.Buffer

	code := run([]string{given}, &stderr)

	if want := "genlex: shadowed: " + given + ":1: digit is defined here and is in the prelude; the definition wins\n"; code != 0 || stderr.String() != want {
		t.Errorf("exit %d, %q; want 0, %q", code, stderr.String(), want)
	}
}

func TestGenlexRefusesADeadRuleAndSaysWhatCoversIt(t *testing.T) {
	given := filepath.Join(t.TempDir(), "d.lex")
	source(t, given, "start {\n  \"a\"  emit a\n  \"a\"  emit b\n}\n")
	var stderr bytes.Buffer

	code := run([]string{given}, &stderr)

	if want := "genlex: " + given + ":3: b is covered in start by a\ngenlex: " + given + ":3: b is covered in every scope it runs in\n"; code != 1 || stderr.String() != want {
		t.Errorf("exit %d, %q; want 1, %q", code, stderr.String(), want)
	}
}

func names(file *ast.File) []string {
	var out []string
	for _, d := range file.Decls {
		if g, ok := d.(*ast.GenDecl); ok && g.Tok != token.IMPORT {
			for _, s := range g.Specs {
				out = append(out, s.(*ast.ValueSpec).Names[0].Name)
			}
		}
	}
	return out
}

func lines(t *testing.T, path, pattern string) []string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	ms := regexp.MustCompile(pattern).FindAllStringSubmatch(string(b), -1)
	out := make([]string, 0, len(ms))
	for _, m := range ms {
		out = append(out, m[1])
	}
	return out
}

func source(t *testing.T, path, src string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestGenlexRefusesAScopeOverItsBudgetOfStates(t *testing.T) {
	given := filepath.Join(t.TempDir(), "x.lex")
	source(t, given, "start {\n  [ab]* \"a\" [ab] [ab] [ab]  emit x\n}\n")
	var stderr bytes.Buffer

	code := run([]string{"-max-states", "8", given}, &stderr)

	if want := "genlex: " + given + ": in start, the patterns need more than 8 states to match together\n"; code != 1 || stderr.String() != want {
		t.Errorf("exit %d, %q; want 1, %q", code, stderr.String(), want)
	}
}
