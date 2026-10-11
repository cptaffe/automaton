package lex_test

import (
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
	s, bad := gen.Spec(flat, ms)
	if len(bad) > 0 {
		t.Fatal(bad)
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
