package lextest

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/cptaffe/automaton/lex"
	"github.com/cptaffe/automaton/lex/gen"
)

type File string

const Default File = "t.lex"

func Compile(t testing.TB, src string) *lex.Spec {
	t.Helper()

	return Default.Compile(t, src)
}

func Error(src string) error { return Default.Error(src) }

func Build(t testing.TB, src string) (*gen.Flat, []gen.Machine) {
	t.Helper()

	return Default.Build(t, src)
}

func (f File) Compile(t testing.TB, src string) *lex.Spec {
	t.Helper()

	spec, err := f.compile(src)

	if err != nil {
		t.Fatalf("compiling %s\n%s\n%v", f, src, err)
	}

	return spec
}

func (f File) Error(src string) error {
	_, err := f.compile(src)

	return err
}

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

	ms, dead, err := gen.Build(flat)

	return flat, ms, dead, err
}

func (f File) compile(src string) (*lex.Spec, error) {
	flat, ms, dead, err := f.build(src)

	if err != nil {
		return nil, err
	}

	if len(dead) > 0 {
		return nil, fmt.Errorf("%s", strings.Join(dead, "\n"))
	}

	return gen.Spec(flat, ms)
}

func Read(t testing.TB, spec *lex.Spec, input string) ([]string, error) {
	t.Helper()

	return ReadFrom(t, spec, spec.Scopes[0].Name, input)
}

func ReadFrom(t testing.TB, spec *lex.Spec, scope, input string) ([]string, error) {
	t.Helper()

	toks := TokensFrom(t, spec, scope, strings.NewReader(input))
	out := make([]string, 0, len(toks))

	for _, tok := range toks {
		if tok.Err != nil {
			return out, tok.Err
		}

		out = append(out, spec.Token(tok.Kind)+"("+tok.Text+")")
	}

	return out, nil
}

func Tokens(t testing.TB, spec *lex.Spec, in io.Reader) []lex.Token {
	t.Helper()

	return TokensFrom(t, spec, spec.Scopes[0].Name, in)
}

func TokensFrom(t testing.TB, spec *lex.Spec, scope string, in io.Reader) []lex.Token {
	t.Helper()

	ch, err := lex.Lex(context.Background(), spec, in, scope)

	if err != nil {
		t.Fatal(err)
	}

	var out []lex.Token

	for tok := range ch {
		out = append(out, tok)

		if tok.Err != nil {
			break
		}
	}

	return out
}
