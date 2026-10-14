package lextest

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/cptaffe/automaton/lex"
	"github.com/cptaffe/automaton/lex/gen"
)

func Compile(t testing.TB, src string) *lex.Spec {
	t.Helper()

	spec, err := compile(src)

	if err != nil {
		t.Fatalf("compiling\n%s\n%v", src, err)
	}

	return spec
}

func Error(src string) error {
	_, err := compile(src)

	return err
}

func compile(src string) (*lex.Spec, error) {
	f, err := gen.Parse(strings.NewReader(src), "t.lex")

	if err != nil {
		return nil, err
	}

	flat, err := gen.Expand(f)

	if err != nil {
		return nil, err
	}

	ms, dead, err := gen.Build(flat)

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

	toks, err := lex.Lex(context.Background(), spec, strings.NewReader(input), spec.Scopes[0].Name)

	if err != nil {
		t.Fatal(err)
	}

	var out []string

	for tok := range toks {
		if tok.Err != nil {
			return out, tok.Err
		}

		out = append(out, spec.Token(tok.Kind)+"("+tok.Text+")")
	}

	return out, nil
}
