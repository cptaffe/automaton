package gen_test

import (
	"reflect"
	"testing"

	"github.com/cptaffe/automaton/lex/lextest"
)

func TestAPatternReadsWhatItSays(t *testing.T) {
	for _, c := range []struct {
		name  string
		given string
		when  string
		then  []string
	}{
		{"a literal", `"ab"  emit x`, "abab", []string{"x(ab)", "x(ab)"}},
		{"an escape in a literal", `"\t\n\\\""  emit x`, "\t\n\\\"", []string{"x(\t\n\\\")"}},
		{"a literal in any case", `"org"i  emit x`, "orgORGoRg", []string{"x(org)", "x(ORG)", "x(oRg)"}},
		{"a literal in any case leaves other runes alone", `"é1"i  emit x`, "é1É1", []string{"x(é1)", "x(É1)"}},
		{"an optional piece", `"a" "b"?  emit x`, "aab", []string{"x(a)", "x(ab)"}},
		{"a repeated piece", `"a" "b"*  emit x`, "abba", []string{"x(abb)", "x(a)"}},
		{"a piece at least once", `"a" "b"+  emit x`, "abab", []string{"x(ab)", "x(ab)"}},
		{"a choice", `"ab" | "c"  emit x`, "abc", []string{"x(ab)", "x(c)"}},
		{"a group", `(lower | "_")+  emit x`, "a_b", []string{"x(a_b)"}},
		{"a class", `[abc]+  emit x`, "cab", []string{"x(cab)"}},
		{"a range", `[a-c]+  emit x`, "cab", []string{"x(cab)"}},
		{"a negated class", "[^a]+  emit x\n  \"a\"  emit a", "bcab", []string{"x(bc)", "a(a)", "x(b)"}},
		{"an escape in a class", `[\]\-\n]+  emit x`, "]-\n", []string{"x(]-\n)"}},
		{"a prelude name", "digit+  emit x", "042", []string{"x(042)"}},
		{"any rune", "any  emit x", "é", []string{"x(é)"}},
		{"what one set leaves of another", "(alpha - [aeiou])+  emit x\n  [aeiou]  emit v", "bcad", []string{"x(bc)", "v(a)", "x(d)"}},
		{"the longest match", "\"if\"  emit kw\n  [a-z]+  emit word", "ifs", []string{"word(ifs)"}},
		{"the first rule of two that read the same", "\"if\"  emit kw\n  [a-z]+  emit word", "if", []string{"kw(if)"}},
		{"ignored text", "[a-z]+  emit word\n  \" \"  ignore", "a b", []string{"word(a)", "word(b)"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			spec := lextest.Compile(t, "start {\n  "+c.given+"\n}\n")

			got, err := lextest.Read(t, spec, c.when)

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
		{"a literal in any case is still that literal", `"org"i  emit x`, "orq", "start: no rule reads 'o'"},
		{"a negated class", "[^a]  emit x", "a", "start: no rule reads 'a'"},
		{"what one set leaves of another", "(alpha - [aeiou])+  emit x", "a", "start: no rule reads 'a'"},
	} {
		t.Run(c.name, func(t *testing.T) {
			spec := lextest.Compile(t, "start {\n  "+c.given+"\n}\n")

			got, err := lextest.Read(t, spec, c.when)

			if err == nil || err.Error() != c.then {
				t.Errorf("read %q, %v; want the error %q", got, err, c.then)
			}
		})
	}
}
