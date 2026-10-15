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
	"github.com/cptaffe/automaton/lex/gen/internal/cases"
)

func TestAScopeDecidesWhichRulesRead(t *testing.T) {
	got, err := lextest.Read(t, cases.Lexers["scopes"], "do do\ndo\n")

	if want := []string{"keyword(do)", "name(do)", "newline(\n)", "keyword(do)", "newline(\n)"}; err != nil || !reflect.DeepEqual(got, want) {
		t.Errorf("read %q, %v; want %q", got, err, want)
	}
}

func TestReadingBeginsInTheFirstScope(t *testing.T) {
	got, err := lextest.Read(t, cases.Lexers["scopesbody"], "do do\ndo\n")

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
		{"spaces on both sides", "diffboth", "abcNXX", []string{"word(abc)", "class(NXX)"}},
		{"no spaces", "diffnone", "abcNXX", []string{"word(abc)", "class(NXX)"}},
		{"a space after", "diffafter", "abcNXX", []string{"word(abc)", "class(NXX)"}},
		{"a space before", "diffbefore", "abcNXX", []string{"word(abc)", "class(NXX)"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			got, err := lextest.Read(t, cases.Lexers[c.given], c.when)

			if err != nil || !reflect.DeepEqual(got, c.then) {
				t.Errorf("read %q, %v; want %q", got, err, c.then)
			}
		})
	}
}

func TestAHyphenInsideANameIsPartOfIt(t *testing.T) {
	got, err := lextest.Read(t, cases.Lexers["hyphen"], "--")

	if want := []string{"x(--)"}; err != nil || !reflect.DeepEqual(got, want) {
		t.Errorf("read %q, %v; want %q", got, err, want)
	}
}

func TestNoRulePointsAtABlankLine(t *testing.T) {
	files, err := filepath.Glob("internal/scan/lex.lex")

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
