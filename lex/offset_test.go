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
	f, err := gen.Parse(strings.NewReader(`start {
  ";" [^\n]*   ignore
  blank+       ignore
  "\n"         emit newline
  [a-z]+       emit word
  "="          emit define
  "+"          emit plus
  "{"          emit lbrace
  "}"          emit rbrace
}
`), "t.lex")
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

			if tok.Start > len(src) || tok.End > len(src) {
				t.Fatalf("%s: %q says it is at %d:%d, and the input is %d long",
					c.name, tok.Text, tok.Start, tok.End, len(src))
			}

			if tok.End-tok.Start != len(tok.Text) {
				t.Errorf("%s: %q is %d long and spans %d:%d",
					c.name, tok.Text, len(tok.Text), tok.Start, tok.End)
			}

			if got := src[tok.Start:tok.End]; got != tok.Text {
				t.Errorf("%s: %d:%d is %q and the token is %q", c.name, tok.Start, tok.End, got, tok.Text)
			}

			if tok.Start < at {
				t.Errorf("%s: %q starts at %d, behind the one before it", c.name, tok.Text, tok.Start)
			}

			at = tok.End
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

		if tok.Text == "word" && src[tok.Start:tok.End] != "word" {
			t.Errorf("the word is at %d:%d, which reads %q", tok.Start, tok.End, src[tok.Start:tok.End])
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
	r, err := lex.Open(spec(t), strings.NewReader("word\n!\n"), "start", "dir/t.lex", 0)

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

	if r.Name() != "t" || r.Source(2) != "!" || r.Source(9) != "" {
		t.Errorf("Name %q, Source(2) %q, Source(9) %q", r.Name(), r.Source(2), r.Source(9))
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
	r, err := lex.Open(spec(t), strings.NewReader("ab cd\n\n  ef\n"), "start", "t.lex", 0)
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
	r, err := lex.Open(spec(t), strings.NewReader("; one\n; two\nword = x\n"), "start", "t.lex", 0)
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
