package gen_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/cptaffe/automaton/internal/lextest"
)

func TestACoveredRuleNamesWhatCoversIt(t *testing.T) {
	for _, c := range []struct {
		name  string
		given string
		then  []string
	}{
		{
			"a rule after an epsilon that reads the same",
			"start {\n  -> words\n  \"if\" emit keyword\n}\n\nwords {\n  [a-z]+ emit word\n}\n",
			[]string{"t.lex:3: keyword is covered in start by word", "t.lex:3: keyword is covered in every scope it runs in"},
		},
		{
			"a rule read only through an epsilon",
			"start {\n  [a-z]+ emit word\n  -> kw\n  \" \" ignore\n}\n\nkw {\n  \"if\" emit keyword\n}\n",
			[]string{"t.lex:8: keyword is covered in every scope it runs in"},
		},
		{
			"a rule covered by two",
			"start {\n  \"a\" -> b\n  [a-z]+ emit word\n  \"if\" emit keyword\n}\n\nb {\n  \"if\" emit keyword\n  \"x\" -> start\n}\n",
			[]string{"t.lex:4: keyword is covered in start by -> b and word", "t.lex:4: keyword is covered in every scope it runs in"},
		},
		{
			"an ignored rule",
			"start {\n  \"x\"  emit x\n  \"x\" ignore\n}\n",
			[]string{"t.lex:3: ignore is covered in start by x", "t.lex:3: ignore is covered in every scope it runs in"},
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			_, ms := lextest.Build(t, c.given)

			var got []string

			for _, m := range ms {
				got = append(got, m.Covered...)
			}

			if err := lextest.Error(c.given); err != nil {
				got = append(got, strings.Split(err.Error(), "\n")...)
			}

			if !reflect.DeepEqual(got, c.then) {
				t.Errorf("said %q; want %q", got, c.then)
			}
		})
	}
}
