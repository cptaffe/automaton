package draw_test

import (
	"strings"
	"testing"

	"github.com/cptaffe/automaton/lex/draw"
	"github.com/cptaffe/automaton/lex/gen"
)

func TestTheDrawingNamesSetsAndScopes(t *testing.T) {
	f, err := gen.Parse(strings.NewReader(`
start {
  "/*"      emit open   -> comment(this)
  lower+    emit name
  blank+    ignore
  [^a-z]    emit other
}

comment(then) {
  "/*"      emit open   -> comment(this)
  "*/"      emit close  -> then
  any       ignore
  -> words
}

words {
  [a-z]+    emit word
}
`), "c.lex")
	if err != nil {
		t.Fatal(err)
	}
	flat, err := gen.Expand(f)
	if err != nil {
		t.Fatal(err)
	}
	ms, _, err := gen.Build(flat)
	if err != nil {
		t.Fatal(err)
	}

	whole, err := draw.Whole(flat, ms).Bytes()
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"lower", "blank", "any - [*/a-z]", "(push start)", "close return", `class="hands"`} {
		if !strings.Contains(string(whole), want) {
			t.Errorf("the drawing does not say %q:\n%s", want, whole)
		}
	}

	shape, err := draw.Shape(flat).Bytes()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(shape), "-> words") {
		t.Errorf("the shape does not show the epsilon:\n%s", shape)
	}

	if got := draw.Summary(ms[0]); !strings.HasPrefix(got, "start, 4 rules, ") {
		t.Errorf("the summary is %q", got)
	}
}
