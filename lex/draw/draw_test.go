package draw_test

import (
	"reflect"
	"testing"

	"github.com/cptaffe/automaton/dot/dottest"
	"github.com/cptaffe/automaton/lex/draw"
	"github.com/cptaffe/automaton/lex/lextest"
)

const comments = `
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
`

func TestTheDrawingShowsEveryScopesMachine(t *testing.T) {
	flat, ms := lextest.Build(t, comments)

	g := draw.Whole(flat, ms)

	if got, want := dottest.Nodes(g), []string{
		"start (start)", "2 other (accept)", "3 ignore (accept)", "4 other (accept)", "5 name (accept)", "6 open (push start) (accept)",
		"comment (start)", "2 ignore (accept)", "3 ignore (accept)", "4 ignore (accept)", "5 ignore (accept)", "6 close return (accept)", "7 open (push comment) (accept)", "8 word (accept)",
	}; !reflect.DeepEqual(got, want) {
		t.Errorf("nodes\n%q\nwant\n%q", got, want)
	}
	if got, want := dottest.Edges(g), []string{
		`start -any - [\t␠/a-z]-> 2 other`, "start -blank-> 3 ignore", "start -/-> 4 other", "start -lower-> 5 name",
		"3 ignore -blank-> 3 ignore", "4 other -*-> 6 open (push start)", "5 name -lower-> 5 name",
		"comment -any - [*/a-z]-> 2 ignore", "comment -*-> 3 ignore", "comment -/-> 4 ignore", "comment -lower-> 5 ignore",
		"3 ignore -/-> 6 close return", "4 ignore -*-> 7 open (push comment)", "5 ignore -lower-> 8 word", "8 word -lower-> 8 word",
		"6 open (push start) --> comment (hands)",
	}; !reflect.DeepEqual(got, want) {
		t.Errorf("edges\n%q\nwant\n%q", got, want)
	}
}

func TestAMoveToAnotherScopeIsDrawnAsAHand(t *testing.T) {
	flat, ms := lextest.Build(t, "start {\n  \"x\"  -> other\n  \"y\"  ignore -> other\n}\n\nother {\n  \"z\"  emit z -> start\n}\n")

	g := draw.Whole(flat, ms)

	if got, want := dottest.Nodes(g), []string{"start (start)", "2 (accept)", "3 ignore (accept)", "other (start)", "2 z (accept)"}; !reflect.DeepEqual(got, want) {
		t.Errorf("nodes %q; want %q", got, want)
	}
	if got, want := dottest.Edges(g), []string{
		"start -x-> 2", "start -y-> 3 ignore", "other -z-> 2 z",
		"2 --> other (hands)", "3 ignore --> other (hands)", "2 z --> start (hands)",
	}; !reflect.DeepEqual(got, want) {
		t.Errorf("edges %q; want %q", got, want)
	}
}

func TestAnEdgeIsLabelledByTheShortestSpelling(t *testing.T) {
	for _, c := range []struct {
		name  string
		given string
		then  []string
	}{
		{"a set the file names", "vowel = [aeiou]\nstart {\n  vowel  emit v\n}\n", []string{"start -vowel-> 2 v"}},
		{"a prelude name", "start {\n  lower  emit v\n}\n", []string{"start -lower-> 2 v"}},
		{"a name less a few runes", "start {\n  alpha - [aeiou]  emit v\n}\n", []string{"start -alpha - [aeiou]-> 2 v"}},
		{"a name less another name", "vowel = [aeiou]\nstart {\n  alpha - vowel  emit v\n}\n", []string{"start -alpha - vowel-> 2 v"}},
		{"runes shorter than a name less one", "start {\n  alpha - [q]  emit v\n}\n", []string{"start -A-Za-pr-z-> 2 v"}},
		{"runes when a name less them is too wide", "start {\n  alpha - [acegikmoqsuwy]  emit v\n}\n", []string{"start -A-Zbdfhjlnprtvxz-> 2 v"}},
		{"too many runes to spell", "start {\n  [acegikmoqsuwyACEGIKMOQSUWY]  emit v\n}\n", []string{"start -ACEGIKMOQSUWYacegikmoqsu and 2 more-> 2 v"}},
		{"too many runes either way", "start {\n  [^acegikmoqsuwyACEGIKMOQSUWY]  emit v\n}\n", []string{"start -\\u0000-@BDFHJLNPRTVXZ-`b and 1114001 more-> 2 v"}},
		{"everything but a rune", "start {\n  [^a]  emit v\n}\n", []string{"start -any - [a]-> 2 v"}},
		{"everything", "start {\n  any  emit v\n}\n", []string{"start -any-> 2 v"}},
		{"a literal", "start {\n  \"ab\"  emit v\n  \"ac\"  emit w\n}\n", []string{"start -a-> 2", "2 -b-> 3 v", "2 -c-> 4 w"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			flat, ms := lextest.Build(t, c.given)

			got := dottest.Edges(draw.Whole(flat, ms))

			if !reflect.DeepEqual(got, c.then) {
				t.Errorf("edges %q; want %q", got, c.then)
			}
		})
	}
}

func TestTheShapeListsEachScopesRules(t *testing.T) {
	flat, _ := lextest.Build(t, "start {\n  \"x\"  -> other\n  -> words\n}\n\nother {\n  \"z\"  emit z -> start\n}\n\nwords {\n  [a-y]  emit w\n}\n")

	g := draw.Shape(flat)

	if got, want := dottest.Nodes(g), []string{
		`start "x" -> other -> words [a-y] emit w (start)`,
		`other "z" emit z -> start`,
	}; !reflect.DeepEqual(got, want) {
		t.Errorf("nodes %q; want %q", got, want)
	}
	if got, want := dottest.Edges(g), []string{
		`start "x" -> other -> words [a-y] emit w --> other "z" emit z -> start (cycle)`,
		`other "z" emit z -> start --> start "x" -> other -> words [a-y] emit w (cycle)`,
	}; !reflect.DeepEqual(got, want) {
		t.Errorf("edges %q; want %q", got, want)
	}
}

func TestASummaryCountsRulesStatesAndClasses(t *testing.T) {
	_, ms := lextest.Build(t, comments)

	got := []string{draw.Summary(ms[0]), draw.Summary(ms[1])}

	if want := []string{"start, 4 rules, 6 states, 5 classes", "comment, 4 rules, 8 states, 5 classes"}; !reflect.DeepEqual(got, want) {
		t.Errorf("the summaries are %q; want %q", got, want)
	}
}
