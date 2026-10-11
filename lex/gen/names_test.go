package gen

import (
	"testing"
	"unicode"

	"github.com/cptaffe/automaton"
)

func TestTheCompatibilityNamesAreTheRestrictedOnes(t *testing.T) {
	for _, c := range []struct {
		name string
		in   []rune
		out  []rune
	}{
		{"digit", []rune("0589"), []rune("٣९ abc")},
		{"alpha", []rune("azAZ"), []rune("0ñ٣ _")},
		{"alnum", []rune("azAZ09"), []rune("ñ_ ")},
		{"word", []rune("azAZ09_"), []rune("ñ -")},
		{"xdigit", []rune("09afAF"), []rune("gG٣")},
		{"punct", []rune("!/:@[`{~"), []rune("a0 ")},
		{"blank", []rune(" \t"), []rune("\n\rA")},
	} {
		got, ok := look(t, c.name)

		if !ok {
			t.Fatalf("%s is not a name", c.name)
		}

		for _, r := range c.in {
			if !got.Has(r) {
				t.Errorf("%s should hold %q", c.name, r)
			}
		}

		for _, r := range c.out {
			if got.Has(r) {
				t.Errorf("%s should not hold %q", c.name, r)
			}
		}
	}
}

func TestTheUnicodeNamesSpanUnicode(t *testing.T) {
	for _, c := range []struct {
		name string
		in   []rune
		out  []rune
	}{
		{"unicode.nd", []rune("0٣९"), []rune("abc ")},
		{"unicode.l", []rune("añα漢"), []rune("0 _")},
		{"unicode.latin", []rune("añ"), []rune("α漢 0")},
		{"unicode.greek", []rune("α"), []rune("a漢")},
		{"unicode.han", []rune("漢"), []rune("aα")},
		{"alphabetic", []rune("añα漢"), []rune("0 _")},
	} {
		got, ok := look(t, c.name)

		if !ok {
			t.Fatalf("%s is not a name", c.name)
		}

		for _, r := range c.in {
			if !got.Has(r) {
				t.Errorf("%s should hold %q", c.name, r)
			}
		}

		for _, r := range c.out {
			if got.Has(r) {
				t.Errorf("%s should not hold %q", c.name, r)
			}
		}
	}
}

func TestIdentifierClassesFollowUax31(t *testing.T) {
	start, _ := look(t, "id-start")
	cont, _ := look(t, "id-continue")

	for _, r := range "aZñα漢" {
		if !start.Has(r) {
			t.Errorf("ID_Start should hold %q", r)
		}
	}

	for _, r := range "0_ -+" {
		if start.Has(r) {
			t.Errorf("ID_Start should not hold %q", r)
		}
	}

	for _, r := range "0٣_a漢" {
		if !cont.Has(r) {
			t.Errorf("ID_Continue should hold %q", r)
		}
	}

	for _, r := range " -+" {
		if cont.Has(r) {
			t.Errorf("ID_Continue should not hold %q", r)
		}
	}
}

func TestOnlyOneSpellingIsAName(t *testing.T) {
	for _, c := range []struct{ ours, theirs string }{
		{"unicode.nd", "unicode.Nd"},
		{"unicode.white-space", "unicode.White_Space"},
		{"unicode.latin", "unicode.Latin"},
		{"unicode.other-alphabetic", "unicode.Other_Alphabetic"},
	} {
		if _, ok := unicodeClass(c.ours); !ok {
			t.Errorf("%s should be a name", c.ours)
		}

		if _, ok := unicodeClass(c.theirs); ok {
			t.Errorf("%s should not be a name; only %s is", c.theirs, c.ours)
		}

		if got := spelling(c.theirs); got != c.ours {
			t.Errorf("%s should point at %s, pointed at %q", c.theirs, c.ours, got)
		}
	}

	if got := spelling("unicode.nonsuch"); got != "" {
		t.Errorf("nonsuch should point nowhere, pointed at %q", got)
	}
}

func TestNoTwoNamesSpellTheSame(t *testing.T) {
	seen := map[string]string{}

	for _, from := range []map[string]*unicode.RangeTable{
		unicode.Categories, unicode.Scripts, unicode.Properties,
	} {
		for name := range from {
			at := kebab(name)

			if was, clash := seen[at]; clash && was != name {
				t.Errorf("%s and %s are different names that are both written %q", was, name, at)
			}

			seen[at] = name
		}
	}

}

func look(t *testing.T, name string) (automaton.Set, bool) {
	t.Helper()

	for _, d := range prelude() {
		if d.Name == name {
			s, ok := d.Expr.Single()

			if !ok {
				t.Fatalf("%s is not one set", name)
			}

			return s, true
		}
	}

	return unicodeClass(name)
}

func TestThePreludeHoldsWhatItPromises(t *testing.T) {
	want := []string{
		"digit", "alpha", "alnum", "lower", "upper", "xdigit", "word",
		"blank", "space", "punct", "graph", "print",
		"alphabetic", "lowercase", "uppercase", "id-start", "id-continue",
	}

	has := map[string]bool{}

	for _, d := range prelude() {
		has[d.Name] = true
	}

	for _, n := range want {
		if !has[n] {
			t.Errorf("the prelude does not define %s", n)
		}
	}
}
