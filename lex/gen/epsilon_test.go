package gen_test

import (
	"strings"
	"sync"
	"testing"

	"github.com/cptaffe/automaton/lex/gen"
)

func refused(t *testing.T, src, want string) {
	t.Helper()
	f, err := gen.Parse(strings.NewReader(src), "x.lex")

	if err == nil {
		_, err = gen.Expand(f)
	}

	if err == nil || !strings.Contains(err.Error(), want) {
		t.Errorf("the refusal is %v; want %q", err, want)
	}
}

func TestAnEpsilonReadsAnotherScopesRulesWhereItStands(t *testing.T) {
	f, err := gen.Parse(strings.NewReader("start {\n  -> words\n  \"if\" emit keyword\n}\n\nwords {\n  [a-z]+ emit word\n}\n"), "x.lex")

	if err != nil {
		t.Fatal(err)
	}

	flat, err := gen.Expand(f)

	if err != nil {
		t.Fatal(err)
	}

	if _, dead, err := gen.Build(flat); err != nil || len(dead) != 1 || !strings.Contains(dead[0], "keyword is covered") {
		t.Errorf("with the epsilon first, the keyword should be dead; got %v %v", dead, err)
	}

	last := build(t, `
start {
  "if"    emit keyword
  -> words
}

words {
  [a-z]+  emit word
}
`)

	if got := strings.Join(kinds(t, last, "if"), " "); got != "keyword(if)" {
		t.Errorf("with the epsilon last, if reads as %q; want keyword(if)", got)
	}
}

func TestAnEpsilonIsChecked(t *testing.T) {
	refused(t, "start {\n  -> nowhere\n}\n", "-> nowhere, and there is no such scope")
	refused(t, "start {\n  \"a\" emit a\n  -> t\n}\n\nt(x) {\n  \"b\" emit b -> x\n}\n", "takes a parameter")
	refused(t, "a {\n  \"a\" emit a\n  -> b\n}\n\nb {\n  -> a\n}\n", "goes round in a loop")
}

func TestADoRidesOnAToken(t *testing.T) {
	refused(t, "start {\n  \"a\" do thing\n}\n", "do thing with nothing emitted")
}

func TestAnErrorIsOnItsOwnLine(t *testing.T) {
	refused(t, "\n\n\nstart {\n  \"a\" emit a\n}\n\nstart {\n  \"b\" emit b\n}\n", "x.lex:8: start is opened twice")
	refused(t, "\nword = nope\n", "x.lex:2: nope is not defined yet")
}

func TestParsingIsSafeAtOnce(t *testing.T) {
	var wg sync.WaitGroup
	errs := make(chan error, 16)

	for range 16 {
		wg.Add(1)

		go func() {
			defer wg.Done()
			_, err := gen.Parse(strings.NewReader("start {\n  digit+ emit number\n}\n"), "x.lex")
			errs <- err
		}()
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		if err != nil {
			t.Error(err)
		}
	}
}

func TestThisInAnIncludedScopeIsTheScopeReadingIt(t *testing.T) {
	spec := build(t, `
start {
  -> common
  "x"  emit x
}

common {
  "("  emit lp -> paren(this)
}

paren(then) {
  "("  emit lp -> paren(this)
  ")"  emit rp -> then
  "y"  emit y
}
`)
	if got := strings.Join(kinds(t, spec, "x((y))x"), " "); got != "x(x) lp(() lp(() y(y) rp()) rp()) x(x)" {
		t.Errorf("the tokens are %q", got)
	}
}

func TestARuleCoveredThroughAnEpsilonIsDead(t *testing.T) {
	f, err := gen.Parse(strings.NewReader("start {\n  [a-z]+ emit word\n  -> kw\n  \" \" ignore\n}\n\nkw {\n  \"if\" emit keyword\n}\n"), "x.lex")
	if err != nil {
		t.Fatal(err)
	}
	flat, err := gen.Expand(f)
	if err != nil {
		t.Fatal(err)
	}
	if _, dead, err := gen.Build(flat); err != nil || len(dead) != 1 || !strings.Contains(dead[0], "keyword is covered") {
		t.Errorf("dead %v, %v", dead, err)
	}
}

func TestARuleErrorIsOnTheRulesLine(t *testing.T) {
	refused(t, "start {\n  \"a\" emit a\n  \"b\" do x\n  \"c\" emit c\n}\n", "x.lex:3: do x with nothing emitted")
	refused(t, "start {\n  \"b\" emit b ignore\n}\n", "x.lex:2: a rule cannot both ignore")
	refused(t, "start {\n  \"a\" emit a\n}\n\nb {\n  \"b\" emit b\n}\n", "x.lex:5: nothing reaches b")
}
