package lextest

import (
	"fmt"
	"io"
	"iter"
	"strings"
	"testing"

	"github.com/cptaffe/automaton/lex"
	"github.com/cptaffe/automaton/lex/gen"
)

type File string

const Default File = "t.lex"

func Error(src string) error { return Default.Error(src) }

func Build(t testing.TB, src string) (*gen.Flat, []gen.Machine) {
	t.Helper()

	return Default.Build(t, src)
}

func (f File) Error(src string) error { return f.compile(src) }

func (f File) Build(t testing.TB, src string) (*gen.Flat, []gen.Machine) {
	t.Helper()

	flat, ms, _, err := f.build(src)

	if err != nil {
		t.Fatalf("building %s\n%s\n%v", f, src, err)
	}

	return flat, ms
}

func (f File) build(src string) (*gen.Flat, []gen.Machine, []string, error) {
	parsed, err := gen.Parse(strings.NewReader(src), string(f))

	if err != nil {
		return nil, nil, nil, err
	}

	flat, err := gen.Expand(parsed)

	if err != nil {
		return nil, nil, nil, err
	}

	ms, dead, err := gen.Build(flat, 0)

	return flat, ms, dead, err
}

func (f File) compile(src string) error {
	_, _, dead, err := f.build(src)

	if err == nil && len(dead) > 0 {
		err = fmt.Errorf("%s", strings.Join(dead, "\n"))
	}

	return err
}

type Lexer struct {
	Lex  func(io.Reader) lex.Tokens[int]
	Name func(int) string
}

func Of[K interface {
	~int
	fmt.Stringer
}](lexf func(io.Reader) lex.Tokens[K]) Lexer {
	run := func(r io.Reader) lex.Tokens[int] { return ints[K]{lexf(r)} }

	return Lexer{Lex: run, Name: func(k int) string { return K(k).String() }}
}

type ints[K ~int] struct{ toks lex.Tokens[K] }

func (t ints[K]) Next() (lex.Token[int], bool) {
	tok, ok := t.toks.Next()

	return lex.Token[int]{Kind: int(tok.Kind), Text: tok.Text, Do: tok.Do, At: tok.At, Err: tok.Err}, ok
}

func (t ints[K]) All() iter.Seq[lex.Token[int]] {
	return func(yield func(lex.Token[int]) bool) {
		for tok, ok := t.Next(); ok && yield(tok); tok, ok = t.Next() {
		}
	}
}

func Read(t testing.TB, lx Lexer, input string) ([]string, error) {
	t.Helper()

	toks := Tokens(t, lx, strings.NewReader(input))
	out := make([]string, 0, len(toks))

	for _, tok := range toks {
		if tok.Err != nil {
			return out, tok.Err
		}

		out = append(out, lx.Name(tok.Kind)+"("+tok.Text+")")
	}

	return out, nil
}

func Tokens(t testing.TB, lx Lexer, in io.Reader) []lex.Token[int] {
	t.Helper()

	var out []lex.Token[int]

	for tok := range lx.Lex(in).All() {
		out = append(out, tok)

		if tok.Err != nil {
			break
		}
	}

	return out
}
