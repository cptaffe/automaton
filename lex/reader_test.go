package lex_test

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/cptaffe/automaton/internal/lextest"
	"github.com/cptaffe/automaton/lex"
	"github.com/cptaffe/automaton/lex/internal/cases"
	"github.com/cptaffe/automaton/lex/internal/cases/does"
)

var errBroken = errors.New("broken")

type broken struct{}

func (broken) Read([]byte) (int, error) { return 0, errBroken }

func TestAnErrorUnwrapsToItsCause(t *testing.T) {
	given := &lex.Error{File: "t.lex", Line: 3, Err: errBroken}

	got := given.Error()

	if want := "t.lex:3: broken"; got != want || !errors.Is(given, errBroken) {
		t.Errorf("the error is %q, which is errBroken: %t; want %q, which is", got, errors.Is(given, errBroken), want)
	}
}

func TestAReadErrorReachesTheCaller(t *testing.T) {
	r := lex.NewReader(cases.Lexers["words"].Lex(lex.Terminated(broken{})), "t.lex")

	_, ok := r.Next()

	if ok || r.Err() == nil || r.Err().Error() != "t.lex:1: broken" || !errors.Is(r.Err(), errBroken) {
		t.Errorf("read %t, %v; want the error t.lex:1: broken, wrapping errBroken", ok, r.Err())
	}
}

func TestTheReaderNamesItsFileWithoutTheExtension(t *testing.T) {
	for _, c := range []struct {
		name  string
		given string
		then  string
	}{
		{"a path", "dir/t.lex", "t"},
		{"no extension", "dir/t", "t"},
		{"two extensions", "t.gen.lex", "t.gen"},
		{"a dot file", ".lex", ".lex"},
	} {
		t.Run(c.name, func(t *testing.T) {
			r := lex.NewReader(cases.Lexers["words"].Lex(lex.Terminated(strings.NewReader(""))), c.given)

			if got := r.Name(); got != c.then || r.From() != c.given {
				t.Errorf("Name %q, From %q; want %q and %q", got, r.From(), c.then, c.given)
			}
		})
	}
}

func TestTheReaderKeepsItsFirstFailure(t *testing.T) {
	r := lex.NewReader(cases.Lexers["words"].Lex(lex.Terminated(strings.NewReader("a\nb\n"))), "t.lex")

	r.Next()
	r.Next()
	r.Fail("first %s", "failure")
	r.FailAt(9, "second")

	if got, want := r.Err().Error(), "t.lex:2: first failure"; got != want || r.Line() != 2 {
		t.Errorf("the error is %q on line %d; want %q on line 2", got, r.Line(), want)
	}
}

func TestTheReaderCountsTheLinesATokenSpans(t *testing.T) {
	r := lex.NewReader(cases.Lexers["spanning"].Lex(lex.Terminated(strings.NewReader("/*\n\n*/\n"))), "t.lex")

	var got []int

	for _, ok := r.Next(); ok; _, ok = r.Next() {
		got = append(got, r.Line())
	}

	if want := []int{3, 4}; !reflect.DeepEqual(got, want) {
		t.Errorf("after each token the line is %v; want %v", got, want)
	}
}

func TestAReturnWithNothingPushedIsAnError(t *testing.T) {
	e := lex.NewEmitter[int, int](strings.NewReader("a"))

	e.Begin()
	_, ok := e.Pop("start")

	var got []string

	if tok, sent := e.Item(); sent {
		got = append(got, fmt.Sprintf("%d:%d %v", tok.At.Line, tok.At.Col, tok.Err))
	}

	if want := []string{"1:1 start: returns with nothing to return to"}; ok || !reflect.DeepEqual(got, want) {
		t.Errorf("popped %t and sent %q; want false and %q", ok, got, want)
	}
}

func TestATokenCarriesItsDo(t *testing.T) {
	lx := cases.Lexers["does"]

	toks := lextest.Tokens(t, lx, strings.NewReader("abc"))

	got := make([]string, 0, len(toks))

	for _, tok := range toks {
		got = append(got, fmt.Sprintf("%s(%s) do %d", lx.Name(tok.Kind), tok.Text, tok.Do))
	}

	if want := []string{"a(a) do 1", "b(b) do 0", "c(c) do 2"}; !reflect.DeepEqual(got, want) {
		t.Errorf("read %q; want %q", got, want)
	}

	if got, want := []int{does.DoFirst, does.DoSecond}, []int{1, 2}; !reflect.DeepEqual(got, want) {
		t.Errorf("the does are %v; want %v", got, want)
	}
}
