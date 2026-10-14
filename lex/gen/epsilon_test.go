package gen_test

import (
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/cptaffe/automaton/internal/lextest"
	"github.com/cptaffe/automaton/lex/gen"
)

func TestAnEpsilonReadsAnotherScopesRulesWhereItStands(t *testing.T) {
	spec := lextest.Compile(t, `
start {
  "if"    emit keyword
  -> words
}

words {
  [a-z]+  emit word
}
`)

	got, err := lextest.Read(t, spec, "if")

	if want := []string{"keyword(if)"}; err != nil || !reflect.DeepEqual(got, want) {
		t.Errorf("read %q, %v; want %q", got, err, want)
	}
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
	spec := lextest.Compile(t, `
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

	got, err := lextest.Read(t, spec, "x((y))x")

	if want := []string{"x(x)", "lp(()", "lp(()", "y(y)", "rp())", "rp())", "x(x)"}; err != nil || !reflect.DeepEqual(got, want) {
		t.Errorf("read %q, %v; want %q", got, err, want)
	}
}
