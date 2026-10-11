package lexgen_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/cptaffe/automaton/lexgen"
	"github.com/cptaffe/automaton/lexrun"
)

func run(t *testing.T, spec *lexrun.Spec, src string) ([]string, int, error) {
	t.Helper()
	l, err := lexrun.New(spec, strings.NewReader(src), spec.Scopes[0].Name)
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
	return out, l.Depth(), l.Err()
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
	got, depth, err := run(t, spec, "a /* b /* c */ d */ e")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"word(a)", "open(/*)", "text( b )", "open(/*)", "text( c )", "close(*/)", "text( d )", "close(*/)", "word(e)"}
	if !reflect.DeepEqual(got, want) || depth != 0 {
		t.Errorf("got %v at depth %d, want %v", got, depth, want)
	}
	if _, depth, _ := run(t, spec, "a /* b /* c"); depth != 2 {
		t.Errorf("an unclosed comment leaves depth %d, want 2", depth)
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
	got, depth, err := run(t, spec, "(x[x(x)x]x)x")
	if err != nil || depth != 0 || len(got) != 12 {
		t.Errorf("got %v at depth %d: %v", got, depth, err)
	}
	if _, _, err := run(t, spec, "(]"); err == nil {
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
	got, depth, err := run(t, spec, "<<!y>>x")
	want := []string{"open(<)", "open(<)", "bang(!)", "y(y)", "close(>)", "close(>)", "x(x)"}
	if err != nil || depth != 0 || !reflect.DeepEqual(got, want) {
		t.Errorf("got %v at depth %d: %v, want %v", got, depth, err, want)
	}
}

func TestATemplateThatDoesNotRecurIsStillCopied(t *testing.T) {
	f, err := lexgen.Parse(strings.NewReader(`
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
	flat, err := lexgen.Expand(f)
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
