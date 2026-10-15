package gen_test

import (
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/cptaffe/automaton/internal/lextest"
	"github.com/cptaffe/automaton/lex/gen"
	"github.com/cptaffe/automaton/lex/gen/internal/cases"
)

func TestAnEpsilonReadsAnotherScopesRulesWhereItStands(t *testing.T) {
	got, err := lextest.Read(t, cases.Lexers["epsilon"], "if")

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
	got, err := lextest.Read(t, cases.Lexers["epsilonthis"], "x((y))x")

	if want := []string{"x(x)", "lp(()", "lp(()", "y(y)", "rp())", "rp())", "x(x)"}; err != nil || !reflect.DeepEqual(got, want) {
		t.Errorf("read %q, %v; want %q", got, err, want)
	}
}
