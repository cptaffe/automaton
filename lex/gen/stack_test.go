package gen_test

import (
	"reflect"
	"testing"

	"github.com/cptaffe/automaton/internal/lextest"
	"github.com/cptaffe/automaton/lex/gen/internal/cases"
)

func TestAParameterPassesThroughATemplateThatDoesNotRecur(t *testing.T) {
	got, err := lextest.Read(t, cases.Lexers["passthrough"], "<!><!>")

	if want := []string{"open(<)", "bang(!)", "close(>)", "open(<)", "bang(!)", "close(>)"}; err != nil || !reflect.DeepEqual(got, want) {
		t.Errorf("read %q, %v; want %q", got, err, want)
	}
}

func TestAStackReturnsToWhereItWasPushed(t *testing.T) {
	for _, c := range []struct {
		name  string
		given string
		when  string
		then  []string
	}{
		{"comments nest", "nested", "a /* b /* c */ d */ e", []string{"word(a)", "open(/*)", "text( b )", "open(/*)", "text( c )", "close(*/)", "text( d )", "close(*/)", "word(e)"}},
		{"two scopes recur through each other", "brackets", "(x[x(x)x]x)x", []string{"lp(()", "x(x)", "lb([)", "x(x)", "lp(()", "x(x)", "rp())", "x(x)", "rb(])", "x(x)", "rp())", "x(x)"}},
		{"a parameter passes through", "passing", "<<!y>>x", []string{"open(<)", "open(<)", "bang(!)", "y(y)", "close(>)", "close(>)", "x(x)"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			got, err := lextest.Read(t, cases.Lexers[c.given], c.when)

			if err != nil || !reflect.DeepEqual(got, c.then) {
				t.Errorf("read %q, %v; want %q", got, err, c.then)
			}
		})
	}
}

func TestAStackRefusesWhatTheScopeOnTopCannotRead(t *testing.T) {
	for _, c := range []struct {
		name  string
		given string
		when  string
		then  string
	}{
		{"an unclosed comment", "nested", "a /* b /* c", "the input ends inside comment"},
		{"a bracket closing a parenthesis", "brackets", "(]", "paren: no rule reads ']'"},
	} {
		t.Run(c.name, func(t *testing.T) {
			_, err := lextest.Read(t, cases.Lexers[c.given], c.when)

			if err == nil || err.Error() != c.then {
				t.Errorf("the error is %v; want %q", err, c.then)
			}
		})
	}
}

func TestATemplateThatDoesNotRecurIsCopiedWithoutAStack(t *testing.T) {
	flat, _ := lextest.Build(t, `
start {
  "a"  -> klass(this)
  "b"  -> other
}

other {
  "c"  -> klass(this)
}

klass(then) {
  "d"  -> then
}
`)

	moves := make([]string, 0, len(flat.Rules))

	for _, r := range flat.Rules {
		moves = append(moves, r.Scope+" "+r.Text+" => "+r.Acts.Goto+" "+r.Acts.Push)
	}

	var copies []string

	for _, s := range flat.Scopes {
		if s.Of == "klass" {
			copies = append(copies, s.Name)
		}
	}

	if want := []string{"klass(start)", "klass(other)"}; !reflect.DeepEqual(copies, want) {
		t.Errorf("klass is copied as %q; want %q", copies, want)
	}

	if want := []string{
		`klass(start) "d" -> then => start `,
		`start "a" -> klass(this) => klass(start) `,
		`klass(other) "d" -> then => other `,
		`other "c" -> klass(this) => klass(other) `,
		`start "b" -> other => other `,
	}; !reflect.DeepEqual(moves, want) {
		t.Errorf("the rules are\n%q\nwant\n%q", moves, want)
	}
}
