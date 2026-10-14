package gen_test

import (
	"reflect"
	"testing"

	"github.com/cptaffe/automaton/lex/lextest"
)

func TestAParameterisedScopeIsCopiedForEachArgument(t *testing.T) {
	flat, _ := lextest.Build(t, `
pattern {
  "["  -> klass(this)
  "a"  emit a  -> clause
}

klass(then) {
  "]"  emit rbrack -> then
  any  ignore
}

clause {
  "b"  emit b  -> klass(clause)
}
`)

	scopes := make([]string, 0, len(flat.Scopes))

	for _, s := range flat.Scopes {
		scopes = append(scopes, s.Name)
	}

	moves := make([]string, 0, len(flat.Rules))

	for _, r := range flat.Rules {
		moves = append(moves, r.Scope+" "+r.Text+" => "+r.Acts.Goto)
	}

	if want := []string{"pattern", "klass(pattern)", "klass(clause)", "clause"}; !reflect.DeepEqual(scopes, want) {
		t.Errorf("the scopes are %q; want %q", scopes, want)
	}

	if want := []string{
		`klass(pattern) "]" emit rbrack -> then => pattern`,
		"klass(pattern) any ignore => ",
		`pattern "[" -> klass(this) => klass(pattern)`,
		`klass(clause) "]" emit rbrack -> then => clause`,
		"klass(clause) any ignore => ",
		`clause "b" emit b -> klass(clause) => klass(clause)`,
		`pattern "a" emit a -> clause => clause`,
	}; !reflect.DeepEqual(moves, want) {
		t.Errorf("the rules are\n%q\nwant\n%q", moves, want)
	}
}
