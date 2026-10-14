package gen_test

import (
	"reflect"
	"testing"

	"github.com/cptaffe/automaton/lex/lextest"
)

const comments = `
start {
  "/*"    emit open -> comment(this)
  [a-z]+  emit word
  " "     ignore
}

comment(then) {
  "/*"     emit open -> comment(this)
  "*/"     emit close -> then
  [a-z ]+  emit text
}
`

const brackets = `
start {
  "("  emit lp -> paren(this)
  "x"  emit x
}

paren(then) {
  "("  emit lp -> paren(this)
  "["  emit lb -> brack(this)
  ")"  emit rp -> then
  "x"  emit x
}

brack(then) {
  "["  emit lb -> brack(this)
  "("  emit lp -> paren(this)
  "]"  emit rb -> then
  "x"  emit x
}
`

const passing = `
start {
  "<"  emit open -> outer(this)
  "x"  emit x
}

outer(then) {
  "<"  emit open -> outer(this)
  "!"  emit bang -> inner(then)
  ">"  emit close -> then
}

inner(then) {
  "y"  emit y
  ">"  emit close -> then
}
`

func TestAParameterPassesThroughATemplateThatDoesNotRecur(t *testing.T) {
	spec := lextest.Compile(t, `
start {
  "<"  emit open -> outer(start)
}

outer(then) {
  "!"  emit bang -> inner(then)
}

inner(then) {
  ">"  emit close -> then
}
`)

	got, err := lextest.Read(t, spec, "<!><!>")

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
		{"comments nest", comments, "a /* b /* c */ d */ e", []string{"word(a)", "open(/*)", "text( b )", "open(/*)", "text( c )", "close(*/)", "text( d )", "close(*/)", "word(e)"}},
		{"two scopes recur through each other", brackets, "(x[x(x)x]x)x", []string{"lp(()", "x(x)", "lb([)", "x(x)", "lp(()", "x(x)", "rp())", "x(x)", "rb(])", "x(x)", "rp())", "x(x)"}},
		{"a parameter passes through", passing, "<<!y>>x", []string{"open(<)", "open(<)", "bang(!)", "y(y)", "close(>)", "close(>)", "x(x)"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			spec := lextest.Compile(t, c.given)

			got, err := lextest.Read(t, spec, c.when)

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
		{"an unclosed comment", comments, "a /* b /* c", "the input ends inside comment"},
		{"a bracket closing a parenthesis", brackets, "(]", "paren: no rule reads ']'"},
	} {
		t.Run(c.name, func(t *testing.T) {
			spec := lextest.Compile(t, c.given)

			_, err := lextest.Read(t, spec, c.when)

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

	if got, want := flat.Instantiations(), []string{"klass(start)", "klass(other)"}; !reflect.DeepEqual(got, want) {
		t.Errorf("instantiations %q; want %q", got, want)
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
