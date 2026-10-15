package gen_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/cptaffe/automaton/internal/lextest"
	"github.com/cptaffe/automaton/lex/gen/internal/cases"
)

func TestABracketClassNamesSetsAsABareWordDoes(t *testing.T) {
	for _, c := range []struct {
		name  string
		given string
		when  string
		then  []string
	}{
		{"a unicode category", "classnd", "12", []string{"x(12)"}},
		{"a prelude name", "classalpha", "ab", []string{"x(ab)"}},
		{"two names", "classtwo", "a1", []string{"x(a1)"}},
		{"a name beside runes", "classrunes", "1_2", []string{"x(1_2)"}},
		{"a name beside a range", "classrange", "9fa0", []string{"x(9fa0)"}},
		{"a name in a negated class", "classnegated", "ab", []string{"x(ab)"}},
		{"a name defined in the file", "classvowel", "ea", []string{"x(ea)"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			got, err := lextest.Read(t, cases.Lexers[c.given], c.when)

			if err != nil || !reflect.DeepEqual(got, c.then) {
				t.Errorf("read %v, %v; want %v", got, err, c.then)
			}
		})
	}
}

func TestANegatedBracketClassLeavesOutTheNamedSet(t *testing.T) {
	got, err := lextest.Read(t, cases.Lexers["notdigit"], "a1")

	if want := []string{"x(a)"}; err == nil || err.Error() != "start: no rule reads '1'" || !reflect.DeepEqual(got, want) {
		t.Errorf("read %q, %v; want %q and no rule reading '1'", got, err, want)
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
