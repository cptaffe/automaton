package gen_test

import (
	"strings"
	"testing"

	"github.com/cptaffe/automaton/lex/gen"
)

func expand(t *testing.T, src string) *gen.Flat {
	t.Helper()
	f, err := gen.Parse(strings.NewReader(src), "x.lex")

	if err != nil {
		t.Fatal(err)
	}

	got, err := gen.Expand(f)

	if err != nil {
		t.Fatal(err)
	}

	return got
}

func TestAParameterisedScopeIsSpecialisedPerUse(t *testing.T) {
	got := expand(t, `
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

	names := make([]string, 0, len(got.Scopes))

	for _, s := range got.Scopes {
		names = append(names, s.Name)
	}

	want := "pattern klass(pattern) klass(clause) clause"

	if joined := strings.Join(names, " "); joined != want {
		t.Errorf("the scopes are %q; want %q", joined, want)
	}

	for _, c := range []struct{ in, to string }{
		{"klass(pattern)", "pattern"},
		{"klass(clause)", "clause"},
	} {
		found := false

		for _, r := range got.Rules {
			if r.Scope == c.in && r.Acts.Emit == "rbrack" {
				found = true

				if r.Acts.Goto != c.to {
					t.Errorf("%s comes out to %s; want %s", c.in, r.Acts.Goto, c.to)
				}
			}
		}

		if !found {
			t.Errorf("%s has no rbrack rule", c.in)
		}
	}
}

func TestATemplateUsedWithNoArgumentIsRefused(t *testing.T) {
	f, err := gen.Parse(strings.NewReader("start {\n  \"a\" -> klass\n}\n\nklass(then) {\n  \"b\" -> then\n}\n"), "x.lex")

	if err != nil {
		t.Fatal(err)
	}

	_, err = gen.Expand(f)

	if err == nil || !strings.Contains(err.Error(), "used without one") {
		t.Errorf("the refusal is %v", err)
	}
}

func TestAPlainScopeGivenAnArgumentIsRefused(t *testing.T) {
	f, err := gen.Parse(strings.NewReader("start {\n  \"a\" -> other(start)\n}\n\nother {\n  \"b\" emit b\n}\n"), "x.lex")

	if err != nil {
		t.Fatal(err)
	}

	_, err = gen.Expand(f)

	if err == nil || !strings.Contains(err.Error(), "takes no parameter") {
		t.Errorf("the refusal is %v", err)
	}
}
