package gen_test

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/cptaffe/automaton/lex"
	"github.com/cptaffe/automaton/lex/gen"
	"github.com/cptaffe/automaton/lex/gen/internal/scan"
)

func read(t *testing.T, src string) []string {
	t.Helper()
	l, err := lex.New(scan.Spec, strings.NewReader(src), "start")

	if err != nil {
		t.Fatal(err)
	}

	var out []string

	for {
		tok, ok := l.Next()

		if !ok {
			break
		}

		out = append(out, scan.Spec.Tokens[tok.Kind]+"("+tok.Text+")")
	}

	if err := l.Err(); err != nil {
		t.Fatalf("the lexer stopped: %v, after %v", err, out)
	}

	return out
}

func holds(got []string, want string) bool {
	for _, g := range got {
		if g == want {
			return true
		}
	}

	return false
}

func TestALineEndsTheWaitForAName(t *testing.T) {
	got := read(t, "start {\n  \"a\"  emit\n  \"b\"  emit x\n}\n")

	if n := count(got, "emit(emit)"); n != 2 {
		t.Errorf("read %d emit keywords, wanted 2; got %v", n, got)
	}

	if holds(got, "name(emit)") {
		t.Errorf("the second emit was read as a name: %v", got)
	}
}

func TestAClassIsNotClosedByALineEnding(t *testing.T) {
	got := read(t, "start {\n  [abc\n  def]  emit x\n}\n")

	if !holds(got, "newline(\n)") {
		t.Errorf("no newline inside the class: %v", got)
	}
}

func TestTheThingsOnlyTheCornersReach(t *testing.T) {
	got := read(t, "start {\n  \"a\"  emit do\n  \"org\"i  emit b\n  [[:unicode.nd:][:alpha:]]  emit c\n}\n")

	for _, want := range []string{
		"emit(emit)",
		"name(do)",
		"anycase(\"org\"i)",
		"class([:unicode.nd:])",
		"class([:alpha:])",
	} {
		if !holds(got, want) {
			t.Errorf("no %s in %v", want, got)
		}
	}
}

func count(got []string, want string) int {
	n := 0

	for _, g := range got {
		if g == want {
			n++
		}
	}

	return n
}

func TestTheLexerLanguageReadsItself(t *testing.T) {
	f, err := os.Open("lex.lex")

	if err != nil {
		t.Fatal(err)
	}

	defer f.Close()
	got, err := gen.Parse(f, "lex.lex")

	if err != nil {
		t.Fatal(err)
	}

	flat, err := gen.Expand(got)

	if err != nil {
		t.Fatal(err)
	}

	ms, dead, err := gen.Build(flat)

	if err != nil {
		t.Fatal(err)
	}

	if len(dead) != 0 {
		t.Fatalf("dead rules in lex.lex: %v", dead)
	}

	want, bad := gen.Spec(flat, ms)

	if len(bad) != 0 {
		t.Fatalf("%v", bad)
	}

	if fmt.Sprint(want) != fmt.Sprint(scan.Spec) {
		t.Error("the table compiled from lex.lex is not the table this package is reading with; run go generate")
	}
}

func TestTheCheckedInTableIsCurrent(t *testing.T) {
	f, err := os.Open("lex.lex")

	if err != nil {
		t.Fatal(err)
	}

	defer f.Close()
	got, err := gen.Parse(f, "lex.lex")

	if err != nil {
		t.Fatal(err)
	}

	flat, err := gen.Expand(got)

	if err != nil {
		t.Fatal(err)
	}

	ms, dead, err := gen.Build(flat)

	if err != nil {
		t.Fatal(err)
	}

	if len(dead) != 0 {
		t.Fatalf("dead rules in lex.lex: %v", dead)
	}

	want, bad := gen.Spec(flat, ms)

	if len(bad) != 0 {
		t.Fatalf("%v", bad)
	}

	if got := strings.Join(scan.Spec.Tokens, " "); got != strings.Join(want.Tokens, " ") {
		t.Errorf("tokens differ; run go generate\n got %s\nwant %s", got, strings.Join(want.Tokens, " "))
	}

	if len(scan.Spec.Ranges) != len(want.Ranges) {
		t.Fatalf("the table has %d ranges and the source compiles to %d; run go generate",
			len(scan.Spec.Ranges), len(want.Ranges))
	}

	for i, r := range scan.Spec.Ranges {
		if r != want.Ranges[i] {
			t.Fatalf("range %d is %v and should be %v; run go generate", i, r, want.Ranges[i])
		}
	}

	if len(scan.Spec.Scopes) != len(want.Scopes) {
		t.Fatalf("the table has %d scopes and the source compiles to %d; run go generate",
			len(scan.Spec.Scopes), len(want.Scopes))
	}
}
