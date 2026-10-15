package gen_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/cptaffe/automaton/internal/lextest"
	"github.com/cptaffe/automaton/lex/gen/internal/cases"
	"github.com/cptaffe/automaton/lex/gen/internal/cases/dos"
)

func TestAPatternReadsWhatItSays(t *testing.T) {
	for _, c := range []struct {
		name  string
		given string
		when  string
		then  []string
	}{
		{"a literal", "patliteral", "abab", []string{"x(ab)", "x(ab)"}},
		{"an escape in a literal", "patescape", "\t\n\\\"", []string{"x(\t\n\\\")"}},
		{"a literal in any case", "patanycase", "orgORGoRg", []string{"x(org)", "x(ORG)", "x(oRg)"}},
		{"a literal in any case leaves other runes alone", "patanycaseother", "é1É1", []string{"x(é1)", "x(É1)"}},
		{"an optional piece", "patoptional", "aab", []string{"x(a)", "x(ab)"}},
		{"a repeated piece", "patstar", "abba", []string{"x(abb)", "x(a)"}},
		{"a piece at least once", "patplus", "abab", []string{"x(ab)", "x(ab)"}},
		{"a choice", "patchoice", "abc", []string{"x(ab)", "x(c)"}},
		{"a group", "patgroup", "a_b", []string{"x(a_b)"}},
		{"a group that opens with a literal", "patgroupliteral", "abcab", []string{"x(abcab)"}},
		{"a group that opens with a class", "patgroupclass", "a_b", []string{"x(a_b)"}},
		{"a group that opens with a group", "patgroupgroup", "abab", []string{"x(abab)"}},
		{"a class", "patclass", "cab", []string{"x(cab)"}},
		{"a range", "patrange", "cab", []string{"x(cab)"}},
		{"a range of one rune", "patrangeone", "aa", []string{"x(aa)"}},
		{"a negated class", "patnegated", "bcab", []string{"x(bc)", "a(a)", "x(b)"}},
		{"an escape in a class", "patclassescape", "]-\n", []string{"x(]-\n)"}},
		{"a prelude name", "patprelude", "042", []string{"x(042)"}},
		{"any rune", "patany", "é", []string{"x(é)"}},
		{"what one set leaves of another", "patdifference", "bcad", []string{"x(bc)", "v(a)", "x(d)"}},
		{"the longest match", "patlongest", "ifs", []string{"word(ifs)"}},
		{"the first rule of two that read the same", "patlongest", "if", []string{"kw(if)"}},
		{"ignored text", "patignored", "a b", []string{"word(a)", "word(b)"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			got, err := lextest.Read(t, cases.Lexers[c.given], c.when)

			if err != nil || !reflect.DeepEqual(got, c.then) {
				t.Errorf("read %q, %v; want %q", got, err, c.then)
			}
		})
	}
}

func TestAPatternRefusesWhatItDoesNotSay(t *testing.T) {
	for _, c := range []struct {
		name  string
		given string
		when  string
		then  string
	}{
		{"a literal in any case is still that literal", "patanycase", "orq", "start: no rule reads 'o'"},
		{"a negated class", "refusenegated", "a", "start: no rule reads 'a'"},
		{"what one set leaves of another", "refusedifference", "a", "start: no rule reads 'a'"},
	} {
		t.Run(c.name, func(t *testing.T) {
			got, err := lextest.Read(t, cases.Lexers[c.given], c.when)

			if err == nil || err.Error() != c.then {
				t.Errorf("read %q, %v; want the error %q", got, err, c.then)
			}
		})
	}
}

func TestADoRidesOnTheTokenWhicheverSideOfTheEmitItIsWritten(t *testing.T) {
	lx := cases.Lexers["dos"]

	toks := lextest.Tokens(t, lx, strings.NewReader("abc"))

	got := make([]string, 0, len(toks))

	for _, tok := range toks {
		got = append(got, fmt.Sprintf("%s(%s) do %d", lx.Name(tok.Kind), tok.Text, tok.Do))
	}

	if want, does := []string{"a(a) do 1", "b(b) do 0", "c(c) do 2"}, []int{dos.DoFirst, dos.DoSecond}; !reflect.DeepEqual(got, want) || !reflect.DeepEqual(does, []int{1, 2}) {
		t.Errorf("read %q with DoFirst, DoSecond = %d; want %q with 1, 2", got, does, want)
	}
}
