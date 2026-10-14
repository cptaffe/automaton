package gen_test

import (
	"strings"
	"testing"

	"github.com/cptaffe/automaton/lex/lextest"
)

func TestASourceIsRefusedWithItsFileLineAndReason(t *testing.T) {
	for _, c := range []struct {
		name  string
		given string
		then  string
	}{
		{"an empty file", "", "rules.lex: no scopes, so there is nowhere to begin"},
		{"a rule outside a scope", "\"a\" emit a\n", "rules.lex:1: \"a\" cannot stand here"},
		{"a scope opened twice", "\n\n\nstart {\n  \"a\" emit a\n}\n\nstart {\n  \"b\" emit b\n}\n", "rules.lex:8: start is opened twice"},
		{"a name defined twice", "x = [a]\nx = [b]\nstart {\n  x emit a\n}\n", "rules.lex:2: x is defined twice"},
		{"a name used before it is defined", "\nword = nope\n", "rules.lex:2: nope is not defined yet; there is alnum, alpha,"},
		{"go's spelling of a class", "start {\n  unicode.Nd emit a\n}\n", "rules.lex:2: unicode.Nd is written unicode.nd here"},
		{"a parameter called this", "start {\n  \"a\" -> t(this)\n}\n\nt(this) {\n  \"b\" emit b\n}\n", "rules.lex:5: a scope cannot take a parameter called this"},
		{"a do with no token", "start {\n  \"a\" emit a\n  \"b\" do x\n  \"c\" emit c\n}\n", "rules.lex:3: do x with nothing emitted: a do rides on a token"},
		{"a pattern with no action", "start {\n  \"a\" emit a\n  \"c\"\n}\n", "a new line cannot stand here"},
		{"a rule that ignores and emits", "start {\n  \"b\" emit b ignore\n}\n", "rules.lex:2: a rule cannot both ignore what it reads and emit it"},
		{"an empty literal", "start {\n  \"\" emit a\n}\n", "rules.lex:2: an empty literal matches nothing, so it would never advance"},
		{"an empty literal in any case", "start {\n  \"\"i emit a\n}\n", "rules.lex:2: an empty literal matches nothing, so it would never advance"},
		{"a literal that runs off its line", "start {\n  \"a emit a\n}\n", "rules.lex:2: a literal runs off the end of its line"},
		{"a range that runs backwards", "start {\n  [z-a] emit a\n}\n", `rules.lex:2: the range "z" to "a" runs backwards`},
		{"a token out of place", "start {\n  \"a\" emit a\n  emit b\n}\n", "rules.lex:3: emit cannot stand here"},
		{"a file that ends inside a rule", "start {\n  \"a\" emit a\n", "rules.lex:3: the file ends in the middle of a rule"},
		{"a difference whose left reads two runes", "word = alpha alpha\nplain = word - alpha\n\nstart {\n  plain emit a\n}\n", "rules.lex:2: the left of a - reads more than one rune"},
		{"a difference whose right reads two runes", "plain = alpha - (alpha alpha)\n\nstart {\n  plain emit a\n}\n", "rules.lex:1: the right of a - reads more than one rune"},
		{"a difference that leaves nothing", "plain = [NX] - alpha\n\nstart {\n  plain emit a\n}\n", "rules.lex:1: what is left reads nothing at all"},
		{"an epsilon to no scope", "start {\n  -> nowhere\n}\n", "rules.lex:2: -> nowhere, and there is no such scope"},
		{"an epsilon to a template", "start {\n  \"a\" emit a\n  -> t\n}\n\nt(x) {\n  \"b\" emit b -> x\n}\n", "rules.lex:3: -> t reads t's rules here, and t takes a parameter"},
		{"epsilons in a loop", "a {\n  \"a\" emit a\n  -> b\n}\n\nb {\n  -> a\n}\n", "rules.lex:7: -> a goes round in a loop back to b"},
		{"a goto to no scope", "start {\n  \"a\" -> nowhere\n}\n", "rules.lex: -> nowhere, and there is no such scope"},
		{"a template used without an argument", "start {\n  \"a\" -> klass\n}\n\nklass(then) {\n  \"b\" -> then\n}\n", "rules.lex:5: klass takes a parameter and was used without one"},
		{"a plain scope given an argument", "start {\n  \"a\" -> other(start)\n}\n\nother {\n  \"b\" emit b\n}\n", "rules.lex:5: other takes no parameter, and start was given one"},
		{"a parameter given an argument", "start {\n  \"a\" -> t(start)\n}\n\nt(then) {\n  \"b\" -> then(start)\n}\n", "rules.lex:6: then is this scope's parameter and cannot take one of its own"},
		{"every scope a template", "t(then) {\n  \"b\" -> then\n}\n", "rules.lex: every scope takes a parameter, so there is nowhere to begin"},
		{"a scope nothing reaches", "start {\n  \"a\" emit a\n}\n\nb {\n  \"b\" emit b\n}\n", "rules.lex:5: nothing reaches b"},
		{"a rule another covers", "start {\n  [a-z]+ emit word\n  \"if\" emit keyword\n}\n", "rules.lex:3: keyword is covered in every scope it runs in"},
	} {
		t.Run(c.name, func(t *testing.T) {
			err := lextest.File("rules.lex").Error(c.given)

			if err == nil || !strings.Contains(err.Error(), c.then) {
				t.Errorf("the refusal is %v; want %q", err, c.then)
			}
		})
	}
}
