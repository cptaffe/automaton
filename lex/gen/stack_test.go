package gen_test

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/cptaffe/automaton/lex"
	"github.com/cptaffe/automaton/lex/gen"
)

func run(t *testing.T, spec *lex.Spec, src string) ([]string, error) {
	t.Helper()
	toks, err := lex.Lex(context.Background(), spec, strings.NewReader(src), spec.Scopes[0].Name)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for tok := range toks {
		if tok.Err != nil {
			return out, tok.Err
		}
		out = append(out, spec.Token(tok.Kind)+"("+tok.Text+")")
	}
	return out, nil
}

func TestCommentsNest(t *testing.T) {
	spec := build(t, `
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
`)
	got, err := run(t, spec, "a /* b /* c */ d */ e")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"word(a)", "open(/*)", "text( b )", "open(/*)", "text( c )", "close(*/)", "text( d )", "close(*/)", "word(e)"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	if _, err := run(t, spec, "a /* b /* c"); err == nil || !strings.Contains(err.Error(), "ends inside comment") {
		t.Errorf("an unclosed comment ends with %v", err)
	}
}

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

func TestMutualRecursion(t *testing.T) {
	spec := build(t, brackets)
	got, err := run(t, spec, "(x[x(x)x]x)x")
	if err != nil || len(got) != 12 {
		t.Errorf("got %v: %v", got, err)
	}
	if _, err := run(t, spec, "(]"); err == nil {
		t.Error("a bracket closing a parenthesis was read")
	}
}

func TestAParameterPassesThrough(t *testing.T) {
	spec := build(t, `
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
`)
	got, err := run(t, spec, "<<!y>>x")
	want := []string{"open(<)", "open(<)", "bang(!)", "y(y)", "close(>)", "close(>)", "x(x)"}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Errorf("got %v: %v, want %v", got, err, want)
	}
}

func TestATemplateThatDoesNotRecurIsStillCopied(t *testing.T) {
	f, err := gen.Parse(strings.NewReader(`
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
`), "t.lex")
	if err != nil {
		t.Fatal(err)
	}
	flat, err := gen.Expand(f)
	if err != nil {
		t.Fatal(err)
	}
	if got := flat.Instantiations(); len(got) != 2 {
		t.Errorf("instantiations %v, want two copies", got)
	}
	for _, r := range flat.Rules {
		if r.Acts.Push != "" || r.Acts.Pop {
			t.Errorf("%s:%d uses the stack", r.Scope, r.Line)
		}
	}
}
