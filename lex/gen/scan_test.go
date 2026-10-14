package gen_test

import (
	"reflect"
	"testing"

	"github.com/cptaffe/automaton/internal/lextest"
	"github.com/cptaffe/automaton/lex/gen/internal/scan"
)

func TestTheScannerSplitsALexFileIntoTokens(t *testing.T) {
	for _, c := range []struct {
		name  string
		given string
		then  []string
	}{
		{
			"a line ends the wait for a name after emit",
			"\"a\"  emit\n\"b\"  emit x\n",
			[]string{`text("a")`, "emit(emit)", "newline(\n)", `text("b")`, "emit(emit)", "name(x)", "newline(\n)"},
		},
		{
			"a line ending inside a class is a token and the class goes on",
			"[abc\ndef]  emit x\n",
			[]string{"lbrack([)", "text(a)", "text(b)", "text(c)", "newline(\n)", "text(d)", "text(e)", "text(f)", "rbrack(])", "emit(emit)", "name(x)", "newline(\n)"},
		},
		{
			"a keyword after emit is a name",
			"\"org\"i  emit do\n",
			[]string{`anycase("org"i)`, "emit(emit)", "name(do)", "newline(\n)"},
		},
		{
			"a class name in brackets is one token",
			"[[:unicode.nd:][:alpha:]]  emit c\n",
			[]string{"lbrack([)", "class([:unicode.nd:])", "class([:alpha:])", "rbrack(])", "emit(emit)", "name(c)", "newline(\n)"},
		},
		{
			"a hyphen inside a word is part of the name and beside one is a minus",
			"a-b = alpha-[x]\n",
			[]string{"name(a-b)", "define(=)", "name(alpha)", "minus(-)", "lbrack([)", "text(x)", "rbrack(])", "newline(\n)"},
		},
		{
			"every action of a rule",
			"\"a\" do act emit x -> next(this)\n",
			[]string{`text("a")`, "do(do)", "name(act)", "emit(emit)", "name(x)", "arrow(->)", "name(next)", "lparen(()", "name(this)", "rparen())", "newline(\n)"},
		},
		{
			"a backslash in a class escapes what follows",
			"[^\\]\\--z]\n",
			[]string{"lbrack([)", "caret(^)", `text(\])`, `text(\-)`, "dash(-)", "text(z)", "rbrack(])", "newline(\n)"},
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			got, err := lextest.Read(t, scan.Spec, c.given)

			if err != nil || !reflect.DeepEqual(got, c.then) {
				t.Errorf("read %q, %v; want %q", got, err, c.then)
			}
		})
	}
}
