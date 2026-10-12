package lex_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"runtime"
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

type endless struct{}

func (endless) Read(p []byte) (int, error) {
	return copy(p, "word\n"), nil
}

type around struct {
	open, close string
	fill        byte
	n           int
}

func (a *around) Read(p []byte) (int, error) {
	switch {
	case a.open != "":
		n := copy(p, a.open)
		a.open = a.open[n:]
		return n, nil
	case a.n > 0:
		n := min(len(p), a.n)
		for i := range n {
			p[i] = a.fill
		}
		a.n -= n
		return n, nil
	case a.close != "":
		n := copy(p, a.close)
		a.close = a.close[n:]
		return n, nil
	}

	return 0, io.EOF
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

func all(t *testing.T, s *lex.Spec, in io.Reader) (toks []lex.Token, last lex.Token) {
	t.Helper()
	ch, err := lex.Lex(context.Background(), s, in, "start")
	if err != nil {
		t.Fatal(err)
	}
	for tok := range ch {
		if tok.Err != nil {
			return toks, tok
		}
		toks = append(toks, tok)
	}
	return toks, lex.Token{}
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
		toks, last := all(t, spec(t), c.in)
		if last.Err != nil {
			t.Fatalf("%s: %v", c.name, last.Err)
		}

		at := 0
		for _, tok := range toks {
			if got := src[tok.At.Offset:tok.End()]; got != tok.Text {
				t.Errorf("%s: %d:%d is %q and the token is %q", c.name, tok.At.Offset, tok.End(), got, tok.Text)
			}
			if tok.At.Offset < at {
				t.Errorf("%s: %q starts at %d, behind the one before it", c.name, tok.Text, tok.At.Offset)
			}
			at = tok.End()
		}
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
	r, err := lex.Open(context.Background(), spec(t), strings.NewReader("word\n!\n"), "start", "dir/t.lex")
	if err != nil {
		t.Fatal(err)
	}
	for _, ok := r.Next(); ok; _, ok = r.Next() {
	}

	var e *lex.Error

	if !errors.As(r.Err(), &e) || e.File != "dir/t.lex" || e.Line != 2 {
		t.Errorf("the error is %v; want dir/t.lex line 2", r.Err())
	}

	if r.Name() != "t" {
		t.Errorf("Name %q; want t", r.Name())
	}
}

func TestPositionsCountFromOne(t *testing.T) {
	toks, _ := all(t, spec(t), strings.NewReader("ab cd\n\n  ef\n"))
	var got []string
	for _, tok := range toks {
		if tok.Text != "\n" {
			got = append(got, fmt.Sprintf("%s@%d:%d", tok.Text, tok.At.Line, tok.At.Col))
		}
	}
	if want := "ab@1:1 cd@1:4 ef@3:3"; strings.Join(got, " ") != want {
		t.Errorf("positions %v; want %s", got, want)
	}
}

func TestIgnoredLinesAreCounted(t *testing.T) {
	toks, _ := all(t, spec(t), strings.NewReader("; one\n; two\nword = x\n"))
	if tok := toks[2]; tok.Text != "word" || tok.At.Line != 3 || tok.At.Col != 1 {
		t.Errorf("%q is at %d:%d; want word at 3:1", tok.Text, tok.At.Line, tok.At.Col)
	}
}

func TestColumnsCountCharacters(t *testing.T) {
	u := specOf(t, "start {\n  [^ \\n]+ emit word\n  \" \" ignore\n  \"\\n\" emit newline\n}\n")
	toks, _ := all(t, u, strings.NewReader("\u00e9\u00e9 ab\n"))
	if tok := toks[1]; tok.Text != "ab" || tok.At.Col != 4 || tok.At.Offset != 5 {
		t.Errorf("ab is %q at column %d, offset %d; want column 4, offset 5", tok.Text, tok.At.Col, tok.At.Offset)
	}
}

func TestTheReaderStreams(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	r, err := lex.Open(ctx, spec(t), endless{}, "start", "t.lex")
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

func TestCancellingStopsTheLexer(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	toks, err := lex.Lex(ctx, spec(t), endless{}, "start")
	if err != nil {
		t.Fatal(err)
	}
	<-toks
	cancel()
	for range toks {
	}
}

func TestAMissingLastNewlineIsSupplied(t *testing.T) {
	for _, in := range []io.Reader{strings.NewReader("a\nb"), &drip{s: "a\nb"}} {
		r, err := lex.Open(context.Background(), spec(t), in, "start", "t.lex")
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

const comments = `start {
  "/*" ([^*] | "*"+ [^*/])* "*"+ "/"  ignore
  blank+  ignore
  "\n"    emit newline
  [a-z]+  emit word
}
`

func TestIgnoredTextIsDropped(t *testing.T) {
	const n = 16 << 20

	for _, c := range []struct {
		name string
		spec string
		in   *around
		col  int
	}{
		{"a run of blanks", "", &around{open: "a", fill: ' ', n: n, close: "b\n"}, n + 2},
		{"a comment", comments, &around{open: "a/*", fill: 'x', n: n, close: "*/b\n"}, n + 6},
	} {
		s := spec(t)
		if c.spec != "" {
			s = specOf(t, c.spec)
		}

		var before, after runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&before)
		toks, last := all(t, s, c.in)
		runtime.ReadMemStats(&after)

		if last.Err != nil {
			t.Fatalf("%s: %v", c.name, last.Err)
		}
		if len(toks) != 3 || toks[1].Text != "b" || toks[1].At.Col != c.col {
			t.Errorf("%s: read %v; want b at column %d", c.name, toks, c.col)
		}
		if grew := after.TotalAlloc - before.TotalAlloc; grew > 1<<20 {
			t.Errorf("%s: reading 16 MiB of ignored text allocated %d bytes", c.name, grew)
		}
	}
}

func TestADroppedMatchThatNeverEndsIsAnError(t *testing.T) {
	toks, last := all(t, specOf(t, comments), &around{open: "a\n/*", fill: 'x', n: 1 << 20})
	if last.Err == nil || len(toks) != 2 {
		t.Fatalf("an unclosed comment read %v, %v", toks, last.Err)
	}
	if !strings.Contains(last.Err.Error(), `'/'`) || last.At.Line != 2 || last.At.Col != 1 {
		t.Errorf("the error is %v at %d:%d; want the / at 2:1", last.Err, last.At.Line, last.At.Col)
	}
}
