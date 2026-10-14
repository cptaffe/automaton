package gen_test

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/cptaffe/automaton/internal/lextest"
	"github.com/cptaffe/automaton/lex/gen"
)

func TestAScopeDecidesWhichRulesRead(t *testing.T) {
	spec := lextest.Compile(t, `
word = alpha+

head {
  "do"    emit keyword  -> body
  word    emit name
  -> blanks
}

body {
  word    emit name
  -> blanks
}

blanks {
  " "     ignore
  "\n"    emit newline  -> head
}
`)

	got, err := lextest.Read(t, spec, "do do\ndo\n")

	if want := []string{"keyword(do)", "name(do)", "newline(\n)", "keyword(do)", "newline(\n)"}; err != nil || !reflect.DeepEqual(got, want) {
		t.Errorf("read %q, %v; want %q", got, err, want)
	}
}

func TestReadingCanBeginInAnyScope(t *testing.T) {
	spec := lextest.Compile(t, `
word = alpha+

head {
  "do"    emit keyword  -> body
  word    emit name
  -> blanks
}

body {
  word    emit name
  -> blanks
}

blanks {
  " "     ignore
  "\n"    emit newline  -> head
}
`)

	got, err := lextest.ReadFrom(t, spec, "body", "do do\ndo\n")

	if want := []string{"name(do)", "name(do)", "newline(\n)", "keyword(do)", "newline(\n)"}; err != nil || !reflect.DeepEqual(got, want) {
		t.Errorf("read %q, %v; want %q", got, err, want)
	}
}

func TestADifferenceReadsWhatIsLeftHoweverItIsSpaced(t *testing.T) {
	for _, c := range []struct {
		name  string
		given string
		when  string
		then  []string
	}{
		{"spaces on both sides", "alpha - [NX]", "abcNXX", []string{"word(abc)", "class(NXX)"}},
		{"no spaces", "alpha-[NX]", "abcNXX", []string{"word(abc)", "class(NXX)"}},
		{"a space after", "alpha- [NX]", "abcNXX", []string{"word(abc)", "class(NXX)"}},
		{"a space before", "alpha -[NX]", "abcNXX", []string{"word(abc)", "class(NXX)"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			spec := lextest.Compile(t, "plain = "+c.given+"\n\nstart {\n  plain+ emit word\n  [NX]+ emit class\n}\n")

			got, err := lextest.Read(t, spec, c.when)

			if err != nil || !reflect.DeepEqual(got, c.then) {
				t.Errorf("read %q, %v; want %q", got, err, c.then)
			}
		})
	}
}

func TestAHyphenInsideANameIsPartOfIt(t *testing.T) {
	spec := lextest.Compile(t, "a = alpha\nb = [b]\na-b = [\\-]\n\nstart {\n  a-b+  emit x\n}\n")

	got, err := lextest.Read(t, spec, "--")

	if want := []string{"x(--)"}; err != nil || !reflect.DeepEqual(got, want) {
		t.Errorf("read %q, %v; want %q", got, err, want)
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

func TestARuleQuotesItsTokensWithoutCommentsOrExtraSpace(t *testing.T) {
	flat, _ := lextest.Build(t, "start {\n  \"x\"    emit y   ; a note\n\n  [a-z]+ -> start\n}\n")

	got := make([]string, 0, len(flat.Rules))

	for _, r := range flat.Rules {
		got = append(got, fmt.Sprintf("%d:%s", r.Line, r.Text))
	}

	if want := []string{`2:"x" emit y`, "4:[a-z]+ -> start"}; !reflect.DeepEqual(got, want) {
		t.Errorf("rules quote %q; want %q", got, want)
	}
}
