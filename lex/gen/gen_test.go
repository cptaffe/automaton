package gen_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cptaffe/automaton/lex"
	"github.com/cptaffe/automaton/lex/gen"
)

func build(t *testing.T, src string) *lex.Spec {
	t.Helper()
	f, err := gen.Parse(strings.NewReader(src), "t.lex")

	if err != nil {
		t.Fatal(err)
	}

	flat, err := gen.Expand(f)

	if err != nil {
		t.Fatal(err)
	}

	ms, dead, err := gen.Build(flat)

	if err != nil || len(dead) != 0 {
		t.Fatalf("%v %v", err, dead)
	}

	spec, bad := gen.Spec(flat, ms)

	if len(bad) != 0 {
		t.Fatalf("%v", bad)
	}

	return spec
}

func kinds(t *testing.T, spec *lex.Spec, src string) []string {
	t.Helper()
	l, err := lex.New(spec, strings.NewReader(src), spec.Scopes[0].Name)

	if err != nil {
		t.Fatal(err)
	}

	var out []string

	for {
		tok, ok := l.Next()

		if !ok {
			break
		}

		out = append(out, spec.Tokens[tok.Kind]+"("+tok.Text+")")
	}

	if err := l.Err(); err != nil {
		t.Fatalf("%v, after %v", err, out)
	}

	return out
}

func TestAScopeIsTheContext(t *testing.T) {
	spec := build(t, `
word = alpha+

head includes blanks {
  "do"    emit keyword  -> body
  word    emit name
}

body includes blanks {
  word    emit name
}

blanks {
  " "     ignore
  "\n"    emit newline  -> head
}
`)

	want := "keyword(do) name(do) newline(\n) keyword(do) newline(\n)"

	if got := strings.Join(kinds(t, spec, "do do\ndo\n"), " "); got != want {
		t.Errorf("the tokens are %q;\nwant %q", got, want)
	}
}

func TestADifferenceNeedsTwoSets(t *testing.T) {
	for _, c := range []struct{ src, want string }{
		{"word = alpha alpha\nplain = word - alpha\n\nstart {\n  plain emit a\n}\n", "left of a - reads more than one rune"},
		{"plain = alpha - (alpha alpha)\n\nstart {\n  plain emit a\n}\n", "right of a - reads more than one rune"},
		{"plain = [NX] - alpha\n\nstart {\n  plain emit a\n}\n", "reads nothing at all"},
	} {
		_, err := gen.Parse(strings.NewReader(c.src), "d.lex")

		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("the refusal is %v; want %s", err, c.want)
		}
	}
}

func TestADifferenceReadsWhatIsLeft(t *testing.T) {
	f, err := gen.Parse(strings.NewReader(
		"plain = alpha - [NX]\n\nstart {\n  plain+ emit word\n  [NX]+ emit class\n}\n"), "d.lex")

	if err != nil {
		t.Fatal(err)
	}

	flat, err := gen.Expand(f)

	if err != nil {
		t.Fatal(err)
	}

	ms, dead, err := gen.Build(flat)

	if err != nil || len(dead) != 0 {
		t.Fatalf("%v %v", err, dead)
	}

	spec, bad := gen.Spec(flat, ms)

	if len(bad) != 0 {
		t.Fatalf("%v", bad)
	}

	if got := strings.Join(kinds(t, spec, "abcNXX"), " "); got != "word(abc) class(NXX)" {
		t.Errorf("abcNXX reads as %q", got)
	}
}

func TestAHyphenInsideANameIsPartOfIt(t *testing.T) {
	for _, src := range []string{
		"a-b = alpha\n\nstart {\n  a-b emit x\n}\n",
		"plain = alpha-[NX]\n\nstart {\n  plain emit x\n}\n",
		"plain = alpha - [NX]\n\nstart {\n  plain emit x\n}\n",
		"plain = alpha- [NX]\n\nstart {\n  plain emit x\n}\n",
		"plain = alpha -[NX]\n\nstart {\n  plain emit x\n}\n",
	} {
		if _, err := gen.Parse(strings.NewReader(src), "h.lex"); err != nil {
			t.Errorf("%q: %v", src, err)
		}
	}
}

func TestTheSpacingOfADifferenceDoesNotMatter(t *testing.T) {
	var want string

	for i, src := range []string{
		"plain = alpha-[NX]\n\nstart {\n  plain+ emit word\n  [NX]+ emit class\n}\n",
		"plain = alpha - [NX]\n\nstart {\n  plain+ emit word\n  [NX]+ emit class\n}\n",
		"plain = alpha- [NX]\n\nstart {\n  plain+ emit word\n  [NX]+ emit class\n}\n",
		"plain = alpha -[NX]\n\nstart {\n  plain+ emit word\n  [NX]+ emit class\n}\n",
	} {
		f, err := gen.Parse(strings.NewReader(src), "h.lex")

		if err != nil {
			t.Fatalf("%q: %v", src, err)
		}

		flat, err := gen.Expand(f)

		if err != nil {
			t.Fatal(err)
		}

		ms, _, err := gen.Build(flat)

		if err != nil {
			t.Fatal(err)
		}

		spec, bad := gen.Spec(flat, ms)

		if len(bad) != 0 {
			t.Fatalf("%v", bad)
		}

		got := strings.Join(kinds(t, spec, "abcNXX"), " ")

		if i == 0 {
			want = got

			if want != "word(abc) class(NXX)" {
				t.Fatalf("abcNXX reads as %q", got)
			}

			continue
		}

		if got != want {
			t.Errorf("%q reads abcNXX as %q; the first spelling gives %q", src, got, want)
		}
	}
}

func TestNoRulePointsAtABlankLine(t *testing.T) {
	files, err := filepath.Glob("lex.lex")

	if err != nil || len(files) == 0 {
		t.Fatalf("no lexers to read: %v", err)
	}

	all := 0

	for _, at := range files {
		b, err := os.ReadFile(at)

		if err != nil {
			t.Fatal(err)
		}

		src := string(b)
		f, err := gen.Parse(strings.NewReader(src), at)

		if err != nil {
			t.Fatalf("%s: %v", at, err)
		}

		lines := strings.Split(src, "\n")

		for _, r := range f.Rules {
			all++

			if r.Line-1 < 0 || r.Line-1 >= len(lines) {
				t.Errorf("%s: a rule says line %d, which the file has not got", at, r.Line)

				continue
			}

			if strings.TrimSpace(lines[r.Line-1]) == "" {
				t.Errorf("%s:%d: a rule points at a blank line", at, r.Line)
			}

			if r.Text == "" {
				t.Errorf("%s:%d: a rule quotes nothing", at, r.Line)
			}
		}
	}

	if all == 0 {
		t.Error("read no rules")
	}
}
