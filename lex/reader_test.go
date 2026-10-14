package lex_test

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/cptaffe/automaton/internal/lextest"
	"github.com/cptaffe/automaton/lex"
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
	r, err := lex.Open(context.Background(), lextest.Compile(t, words), broken{}, "start", "t.lex")

	if err != nil {
		t.Fatal(err)
	}

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
			r, err := lex.Open(context.Background(), lextest.Compile(t, words), strings.NewReader(""), "start", c.given)

			if err != nil {
				t.Fatal(err)
			}

			if got := r.Name(); got != c.then || r.From() != c.given {
				t.Errorf("Name %q, From %q; want %q and %q", got, r.From(), c.then, c.given)
			}
		})
	}
}

func TestTheReaderKeepsItsFirstFailure(t *testing.T) {
	r, err := lex.Open(context.Background(), lextest.Compile(t, words), strings.NewReader("a\nb\n"), "start", "t.lex")

	if err != nil {
		t.Fatal(err)
	}

	r.Next()
	r.Next()
	r.Fail("first %s", "failure")
	r.FailAt(9, "second")

	if got, want := r.Err().Error(), "t.lex:2: first failure"; got != want || r.Line() != 2 {
		t.Errorf("the error is %q on line %d; want %q on line 2", got, r.Line(), want)
	}
}

func TestTheReaderCountsTheLinesATokenSpans(t *testing.T) {
	spec := lextest.Compile(t, "start {\n  \"/*\" [^*]* \"*/\"  emit comment\n  \"\\n\"  emit newline\n}\n")

	r, err := lex.Open(context.Background(), spec, strings.NewReader("/*\n\n*/\n"), "start", "t.lex")

	if err != nil {
		t.Fatal(err)
	}

	var got []int

	for _, ok := r.Next(); ok; _, ok = r.Next() {
		got = append(got, r.Line())
	}

	if want := []int{3, 4}; !reflect.DeepEqual(got, want) {
		t.Errorf("after each token the line is %v; want %v", got, want)
	}
}

func TestLexingFromAnUnknownScopeIsRefused(t *testing.T) {
	_, err := lex.Lex(context.Background(), lextest.Compile(t, words), strings.NewReader(""), "nowhere")

	if err == nil || err.Error() != "no scope called nowhere" {
		t.Errorf("the error is %v; want no scope called nowhere", err)
	}

	_, err = lex.Open(context.Background(), lextest.Compile(t, words), strings.NewReader(""), "nowhere", "t.lex")

	if err == nil || err.Error() != "no scope called nowhere" {
		t.Errorf("Open's error is %v; want no scope called nowhere", err)
	}
}

func TestAReturnWithNothingPushedIsAnError(t *testing.T) {
	given := &lex.Spec{
		Tokens: []string{"a"},
		Ranges: []lex.Range{{Lo: 'a', Hi: 'a', Class: 1}},
		Scopes: []lex.Scope{{
			Name:   "start",
			Start:  1,
			States: []lex.State{{Next: []int{0, 0}}, {Next: []int{0, 2}}, {Next: []int{0, 0}, Accept: 1}},
			Acts:   []lex.Act{{Emit: 1, Pop: true}},
		}},
	}

	got, err := lextest.Read(t, given, "a")

	if err == nil || err.Error() != "start: returns with nothing to return to" || len(got) != 0 {
		t.Errorf("read %q, %v; want nothing and start: returns with nothing to return to", got, err)
	}
}

func TestATokenCarriesItsDo(t *testing.T) {
	spec := lextest.Compile(t, "start {\n  \"a\"  do first emit a\n  \"b\"  emit b\n  \"c\"  emit c do second\n}\n")

	toks := lextest.Tokens(t, spec, strings.NewReader("abc"))

	got := make([]string, 0, len(toks))

	for _, tok := range toks {
		got = append(got, fmt.Sprintf("%s(%s) do %d", spec.Token(tok.Kind), tok.Text, tok.Do))
	}

	if want := []string{"a(a) do 1", "b(b) do 0", "c(c) do 2"}; !reflect.DeepEqual(got, want) {
		t.Errorf("read %q; want %q", got, want)
	}

	if want := []string{"first", "second"}; !reflect.DeepEqual(spec.Does, want) {
		t.Errorf("the does are %q; want %q", spec.Does, want)
	}
}
