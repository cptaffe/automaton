package lextest_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/cptaffe/automaton/internal/lextest"
)

const scopes = "start {\n  \"a\"  emit a -> other\n}\n\nother {\n  \"b\"  emit b -> start\n}\n"

func TestReadStartsInTheFirstScope(t *testing.T) {
	spec := lextest.Compile(t, scopes)

	got, err := lextest.Read(t, spec, "abab")

	if want := []string{"a(a)", "b(b)", "a(a)", "b(b)"}; err != nil || !reflect.DeepEqual(got, want) {
		t.Errorf("read %q, %v; want %q", got, err, want)
	}
}

func TestReadFromStartsInTheScopeNamed(t *testing.T) {
	spec := lextest.Compile(t, scopes)

	got, err := lextest.ReadFrom(t, spec, "other", "bab")

	if want := []string{"b(b)", "a(a)", "b(b)"}; err != nil || !reflect.DeepEqual(got, want) {
		t.Errorf("read %q, %v; want %q", got, err, want)
	}
}

func TestReadStopsAtTheFirstError(t *testing.T) {
	spec := lextest.Compile(t, scopes)

	got, err := lextest.Read(t, spec, "aa")

	if want := []string{"a(a)"}; err == nil || err.Error() != "other: no rule reads 'a'" || !reflect.DeepEqual(got, want) {
		t.Errorf("read %q, %v; want %q and other: no rule reads 'a'", got, err, want)
	}
}

func TestAnErrorNamesTheFile(t *testing.T) {
	for _, c := range []struct {
		name  string
		given lextest.File
		then  string
	}{
		{"the default file", lextest.Default, "t.lex:2: x is not defined yet"},
		{"a file the test names", "dir/rules.lex", "dir/rules.lex:2: x is not defined yet"},
	} {
		t.Run(c.name, func(t *testing.T) {
			err := c.given.Error("start {\n  x  emit a\n}\n")

			if err == nil || !strings.HasPrefix(err.Error(), c.then) {
				t.Errorf("the error is %v; want it to begin %q", err, c.then)
			}
		})
	}
}

func TestADeadRuleIsAnError(t *testing.T) {
	given := "start {\n  \"a\"  emit a\n  \"a\"  emit b\n}\n"

	err := lextest.Error(given)

	if want := "t.lex:3: b is covered in every scope it runs in"; err == nil || err.Error() != want {
		t.Errorf("the error is %v; want %q", err, want)
	}
}
