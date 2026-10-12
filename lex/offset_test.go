package lex_test

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/cptaffe/automaton/lex"
	"github.com/cptaffe/automaton/lex/gen"
)

type drip struct {
	s  string
	at int
}

func (d *drip) Read(p []byte) (int, error) {
	if d.at >= len(d.s) {
		return 0, io.EOF
	}

	p[0] = d.s[d.at]
	d.at++

	return 1, nil
}

func spec(t *testing.T) *lex.Spec {
	t.Helper()
	return specOf(t, `start {
  ";" [^\n]*   ignore
  blank+       ignore
  "\n"         emit newline
  [a-z]+       emit word
  "="          emit define
  "+"          emit plus
  "{"          emit lbrace
  "}"          emit rbrace
}
`)
}

func specOf(t *testing.T, src string) *lex.Spec {
	t.Helper()
	f, err := gen.Parse(strings.NewReader(src), "t.lex")
	if err != nil {
		t.Fatal(err)
	}
	flat, err := gen.Expand(f)
	if err != nil {
		t.Fatal(err)
	}
	ms, _, err := gen.Build(flat)
	if err != nil {
		t.Fatal(err)
	}
	s, err := gen.Spec(flat, ms)

	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestWhereATokenIsDoesNotDependOnHowItArrived(t *testing.T) {
	src := "word = alpha+  ; a comment\nstart {\n  word  emit word\n}\n"

	for _, c := range []struct {
		name string
		in   io.Reader
	}{
		{"all at once", strings.NewReader(src)},
		{"a byte at a time", &drip{s: src}},
	} {
		l, err := lex.New(spec(t), c.in, "start")

		if err != nil {
			t.Fatal(err)
		}

		at := 0

		for {
			tok, ok := l.Next()

			if !ok {
				break
			}

			if tok.At.Offset > len(src) || tok.End() > len(src) {
				t.Fatalf("%s: %q says it is at %d:%d, and the input is %d long",
					c.name, tok.Text, tok.At.Offset, tok.End(), len(src))
			}

			if got := src[tok.At.Offset:tok.End()]; got != tok.Text {
				t.Errorf("%s: %d:%d is %q and the token is %q", c.name, tok.At.Offset, tok.End(), got, tok.Text)
			}

			if tok.At.Offset < at {
				t.Errorf("%s: %q starts at %d, behind the one before it", c.name, tok.Text, tok.At.Offset)
			}

			at = tok.End()
		}

		if err := l.Err(); err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
	}
}

func TestALongRunOfIgnoredTextNeedsNoBuffer(t *testing.T) {
	src := strings.Repeat("; "+strings.Repeat("x", 1000)+"\n", 3000) + "word\n"
	l, err := lex.New(spec(t), strings.NewReader(src), "start")

	if err != nil {
		t.Fatal(err)
	}

	var kinds []string

	for {
		tok, ok := l.Next()

		if !ok {
			break
		}

		if tok.Text != "\n" {
			kinds = append(kinds, tok.Text)
		}

		if tok.Text == "word" && src[tok.At.Offset:tok.End()] != "word" {
			t.Errorf("the word is at %d:%d, which reads %q", tok.At.Offset, tok.End(), src[tok.At.Offset:tok.End()])
		}
	}

	if err := l.Err(); err != nil {
		t.Fatalf("three megabytes of comment: %v", err)
	}

	if strings.Join(kinds, " ") != "word" {
		t.Errorf("the tokens are %v; want only the word", kinds)
	}
}

func TestATokenNamesItsKind(t *testing.T) {
	s := spec(t)

	if s.Token(0) != "" {
		t.Errorf("kind 0 is %q; it is no token", s.Token(0))
	}

	if s.Token(1) != s.Tokens[0] || s.Token(len(s.Tokens)+1) != "" {
		t.Error("kinds count from 1, and none lies past the last")
	}
}

func TestAnErrorCarriesItsPlace(t *testing.T) {
	r, err := lex.Open(spec(t), strings.NewReader("word\n!\n"), "start", "dir/t.lex")

	if err != nil {
		t.Fatal(err)
	}

	for {
		if _, ok := r.Next(); !ok {
			break
		}
	}

	var e *lex.Error

	if !errors.As(r.Err(), &e) || e.File != "dir/t.lex" || e.Line != 2 {
		t.Errorf("the error is %v; want dir/t.lex line 2", r.Err())
	}

	if r.Name() != "t" {
		t.Errorf("Name %q; want t", r.Name())
	}
}

func TestOneIgnoredMatchLongerThanAMegabyte(t *testing.T) {
	src := "a" + strings.Repeat(" ", 2<<20) + "b\n"
	l, err := lex.New(spec(t), strings.NewReader(src), "start")
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for tok, ok := l.Next(); ok; tok, ok = l.Next() {
		got = append(got, tok.Text)
	}
	if err := l.Err(); err != nil || strings.Join(got, "") != "ab\n" {
		t.Errorf("read %q, %v", got, err)
	}
}

func TestPositionsCountFromOne(t *testing.T) {
	r, err := lex.Open(spec(t), strings.NewReader("ab cd\n\n  ef\n"), "start", "t.lex")
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for tok, ok := r.Next(); ok; tok, ok = r.Next() {
		if tok.Text != "\n" {
			got = append(got, fmt.Sprintf("%s@%d:%d", tok.Text, tok.At.Line, tok.At.Col))
		}
	}
	if want := "ab@1:1 cd@1:4 ef@3:3"; strings.Join(got, " ") != want {
		t.Errorf("positions %v; want %s", got, want)
	}
}

func TestIgnoredLinesAreCounted(t *testing.T) {
	r, err := lex.Open(spec(t), strings.NewReader("; one\n; two\nword = x\n"), "start", "t.lex")
	if err != nil {
		t.Fatal(err)
	}
	tok, _ := r.Next()
	for tok.Text != "word" {
		tok, _ = r.Next()
	}
	if tok.At.Line != 3 || tok.At.Col != 1 {
		t.Errorf("word is at %d:%d; want 3:1", tok.At.Line, tok.At.Col)
	}
}

type endless struct{}

func (endless) Read(p []byte) (int, error) {
	n := copy(p, "word\n")

	return n, nil
}

func TestTheReaderStreams(t *testing.T) {
	r, err := lex.Open(spec(t), endless{}, "start", "t.lex")
	if err != nil {
		t.Fatal(err)
	}
	for i := range 1000 {
		tok, ok := r.Next()
		if !ok {
			t.Fatalf("the input ended after %d tokens: %v", i, r.Err())
		}
		if want := i/2 + 1; tok.At.Line != want {
			t.Fatalf("%q is on line %d; want %d", tok.Text, tok.At.Line, want)
		}
	}
}

func TestAMissingLastNewlineIsSupplied(t *testing.T) {
	for _, in := range []io.Reader{strings.NewReader("a\nb"), &drip{s: "a\nb"}} {
		r, err := lex.Open(spec(t), in, "start", "t.lex")
		if err != nil {
			t.Fatal(err)
		}
		var got []string
		for tok, ok := r.Next(); ok; tok, ok = r.Next() {
			got = append(got, fmt.Sprintf("%q@%d:%d", tok.Text, tok.At.Line, tok.At.Col))
		}
		if want := `"a"@1:1 "\n"@1:2 "b"@2:1 "\n"@2:2`; strings.Join(got, " ") != want || r.Err() != nil {
			t.Errorf("read %v, %v; want %s", got, r.Err(), want)
		}
	}
}

func TestColumnsCountCharacters(t *testing.T) {
	u := specOf(t, "start {\n  [^ \\n]+ emit word\n  \" \" ignore\n  \"\\n\" emit newline\n}\n")
	l, err := lex.New(u, strings.NewReader("\u00e9\u00e9 ab\n"), "start")
	if err != nil {
		t.Fatal(err)
	}
	l.Next()
	if tok, _ := l.Next(); tok.Text != "ab" || tok.At.Col != 4 || tok.At.Offset != 5 {
		t.Errorf("ab is %q at column %d, offset %d; want column 4, offset 5", tok.Text, tok.At.Col, tok.At.Offset)
	}
}
