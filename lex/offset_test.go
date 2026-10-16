package lex_test

import (
	"errors"
	"fmt"
	"io"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/cptaffe/automaton/internal/lextest"
	"github.com/cptaffe/automaton/lex"
	"github.com/cptaffe/automaton/lex/internal/cases"
	"github.com/cptaffe/automaton/lex/internal/cases/words"
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

func positions(toks []lex.Token[int]) []string {
	out := make([]string, 0, len(toks))

	for _, tok := range toks {
		out = append(out, fmt.Sprintf("%q@%d:%d", tok.Text, tok.At.Line, tok.At.Col))
	}

	return out
}

func TestATokensPlaceDoesNotDependOnHowTheInputArrived(t *testing.T) {
	src := "word = alpha+  ; a comment\nstart {\n  word  emit word\n}\n"
	want := []string{
		"word@0:1:1", "define@5:1:6", "word@7:1:8", "plus@12:1:13", "newline@26:1:27",
		"word@27:2:1", "lbrace@33:2:7", "newline@34:2:8",
		"word@37:3:3", "word@43:3:9", "word@48:3:14", "newline@52:3:18",
		"rbrace@53:4:1", "newline@54:4:2",
	}

	for _, c := range []struct {
		name  string
		given io.Reader
		then  []string
	}{
		{"all at once", strings.NewReader(src), want},
		{"a byte at a time", &drip{s: src}, want},
	} {
		t.Run(c.name, func(t *testing.T) {
			lx := cases.Lexers["words"]

			toks := lextest.Tokens(t, lx, c.given)

			got := make([]string, 0, len(toks))

			for _, tok := range toks {
				got = append(got, fmt.Sprintf("%s@%d:%d:%d", lx.Name(tok.Kind), tok.At.Offset, tok.At.Line, tok.At.Col))

				if src[tok.At.Offset:tok.End()] != tok.Text {
					t.Errorf("%q at %d is not what the source holds there", tok.Text, tok.At.Offset)
				}
			}

			if !reflect.DeepEqual(got, c.then) {
				t.Errorf("read\n%q\nwant\n%q", got, c.then)
			}
		})
	}
}

func TestAKindNumbersTheSortedTokensFromOne(t *testing.T) {
	got := []string{words.Tok(-1).String(), words.Tok(0).String(), words.Tok(1).String(), words.Tok(6).String(), words.Tok(7).String()}

	if want := []string{"", "", "define", "word", ""}; !reflect.DeepEqual(got, want) {
		t.Errorf("kinds -1, 0, 1, 6 and 7 are %q; want %q", got, want)
	}
}

func TestAnErrorCarriesItsFileAndLine(t *testing.T) {
	r := lex.NewReader(cases.Lexers["words"].Lex(lex.Terminated(strings.NewReader("word\n!\n"))), "dir/t.lex")

	for _, ok := r.Next(); ok; _, ok = r.Next() {
	}

	var e *lex.Error

	if !errors.As(r.Err(), &e) || e.File != "dir/t.lex" || e.Line != 2 || r.Err().Error() != `dir/t.lex:2: start: no rule reads '!'` {
		t.Errorf("the error is %v; want dir/t.lex:2: start: no rule reads '!'", r.Err())
	}
}

func TestLinesAndColumnsCountFromOne(t *testing.T) {
	got := positions(lextest.Tokens(t, cases.Lexers["words"], strings.NewReader("ab cd\n\n  ef\n")))

	if want := []string{`"ab"@1:1`, `"cd"@1:4`, `"\n"@1:6`, `"\n"@2:1`, `"ef"@3:3`, `"\n"@3:5`}; !reflect.DeepEqual(got, want) {
		t.Errorf("positions %q; want %q", got, want)
	}
}

func TestIgnoredLinesAreCounted(t *testing.T) {
	got := positions(lextest.Tokens(t, cases.Lexers["words"], strings.NewReader("; one\n; two\nword = x\n")))

	if want := []string{`"\n"@1:6`, `"\n"@2:6`, `"word"@3:1`, `"="@3:6`, `"x"@3:8`, `"\n"@3:9`}; !reflect.DeepEqual(got, want) {
		t.Errorf("positions %q; want %q", got, want)
	}
}

func TestColumnsCountCharactersAndOffsetsCountBytes(t *testing.T) {
	toks := lextest.Tokens(t, cases.Lexers["unspaced"], strings.NewReader("\u00e9\u00e9 ab\n"))

	got := make([]string, 0, len(toks))

	for _, tok := range toks {
		got = append(got, fmt.Sprintf("%q@%d:%d:%d", tok.Text, tok.At.Offset, tok.At.Line, tok.At.Col))
	}

	if want := []string{`"éé"@0:1:1`, `"ab"@5:1:4`, `"\n"@7:1:6`}; !reflect.DeepEqual(got, want) {
		t.Errorf("positions %q; want %q", got, want)
	}
}

func TestTheReaderStreamsAnEndlessInput(t *testing.T) {
	r := lex.NewReader(cases.Lexers["words"].Lex(lex.Terminated(endless{})), "t.lex")

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

func TestStoppingARangeLeavesTheRestToRead(t *testing.T) {
	toks := words.Lex(endless{})

	for range toks.All() {
		break
	}

	if tok, ok := toks.Next(); !ok || tok.Text != "\n" {
		t.Errorf("after one token Next gives %q, %t; want the newline", tok.Text, ok)
	}
}

func TestTheReaderSuppliesAMissingLastNewline(t *testing.T) {
	for _, c := range []struct {
		name  string
		given io.Reader
		then  []string
	}{
		{"all at once", strings.NewReader("a\nb"), []string{`"a"@1:1`, `"\n"@1:2`, `"b"@2:1`, `"\n"@2:2`}},
		{"a byte at a time", &drip{s: "a\nb"}, []string{`"a"@1:1`, `"\n"@1:2`, `"b"@2:1`, `"\n"@2:2`}},
		{"a last newline already there", strings.NewReader("a\n"), []string{`"a"@1:1`, `"\n"@1:2`}},
		{"nothing at all", strings.NewReader(""), nil},
	} {
		t.Run(c.name, func(t *testing.T) {
			r := lex.NewReader(cases.Lexers["words"].Lex(lex.Terminated(c.given)), "t.lex")

			var got []string

			for tok, ok := r.Next(); ok; tok, ok = r.Next() {
				got = append(got, fmt.Sprintf("%q@%d:%d", tok.Text, tok.At.Line, tok.At.Col))
			}

			if !reflect.DeepEqual(got, c.then) || r.Err() != nil {
				t.Errorf("read %q, %v; want %q", got, r.Err(), c.then)
			}
		})
	}
}

func TestIgnoredTextIsDroppedAsItIsRead(t *testing.T) {
	const n = 16 << 20

	for _, c := range []struct {
		name  string
		given lextest.Lexer
		when  *around
		then  []string
	}{
		{"a run of blanks", cases.Lexers["words"], &around{open: "a", fill: ' ', n: n, close: "b\n"}, []string{`"a"@1:1`, fmt.Sprintf(`"b"@1:%d`, n+2), fmt.Sprintf(`"\n"@1:%d`, n+3)}},
		{"a comment", cases.Lexers["comments"], &around{open: "a/*", fill: 'x', n: n, close: "*/b\n"}, []string{`"a"@1:1`, fmt.Sprintf(`"b"@1:%d`, n+6), fmt.Sprintf(`"\n"@1:%d`, n+7)}},
	} {
		t.Run(c.name, func(t *testing.T) {
			var before, after runtime.MemStats

			runtime.GC()
			runtime.ReadMemStats(&before)

			toks := lextest.Tokens(t, c.given, c.when)

			runtime.ReadMemStats(&after)

			if got := positions(toks); !reflect.DeepEqual(got, c.then) {
				t.Errorf("read %q; want %q", got, c.then)
			}

			if grew := after.TotalAlloc - before.TotalAlloc; grew > 1<<20 {
				t.Errorf("reading 16 MiB of ignored text allocated %d bytes", grew)
			}
		})
	}
}

func TestAnIgnoredMatchThatNeverEndsIsAnErrorWhereItBegan(t *testing.T) {
	toks := lextest.Tokens(t, cases.Lexers["comments"], &around{open: "a\n/*", fill: 'x', n: 1 << 20})

	got := make([]string, 0, len(toks))

	for _, tok := range toks {
		got = append(got, fmt.Sprintf("%q@%d:%d %v", tok.Text, tok.At.Line, tok.At.Col, tok.Err))
	}

	if want := []string{`"a"@1:1 <nil>`, `"\n"@1:2 <nil>`, `""@2:1 start: no rule reads '/'`}; !reflect.DeepEqual(got, want) {
		t.Errorf("read %q; want %q", got, want)
	}
}
