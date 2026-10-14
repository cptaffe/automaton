package gen_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/cptaffe/automaton/lex/lextest"
)

func TestABracketClassNamesSetsAsABareWordDoes(t *testing.T) {
	for _, c := range []struct {
		name  string
		given string
		when  string
		then  []string
	}{
		{"a unicode category", "[[:unicode.nd:]]+", "12", []string{"x(12)"}},
		{"a prelude name", "[[:alpha:]]+", "ab", []string{"x(ab)"}},
		{"two names", "[[:alpha:][:digit:]]+", "a1", []string{"x(a1)"}},
		{"a name beside runes", "[[:digit:]_]+", "1_2", []string{"x(1_2)"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			spec := lextest.Compile(t, "start {\n  "+c.given+"  emit x\n}\n")

			got, err := lextest.Read(t, spec, c.when)

			if err != nil || !reflect.DeepEqual(got, c.then) {
				t.Errorf("read %v, %v; want %v", got, err, c.then)
			}
		})
	}
}

func TestABracketClassRefusesWhatIsNotASetOfRunes(t *testing.T) {
	for _, c := range []struct {
		name  string
		given string
		then  string
	}{
		{"go's spelling", "[[:Nd:]]", "Nd is written unicode.nd here"},
		{"an unknown name", "[[:nope:]]", "nope is not defined yet"},
		{"a pattern", "[[:word:]]", "[:word:] is not a set of runes"},
	} {
		t.Run(c.name, func(t *testing.T) {
			err := lextest.Error("word = alpha alpha\nstart {\n  " + c.given + "  emit x\n}\n")

			if err == nil || !strings.Contains(err.Error(), c.then) {
				t.Errorf("got %v; want an error saying %q", err, c.then)
			}
		})
	}
}
