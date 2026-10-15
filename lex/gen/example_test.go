package gen_test

import (
	"os"
	"reflect"
	"slices"
	"testing"

	"github.com/cptaffe/automaton/internal/lextest"
	"github.com/cptaffe/automaton/lex/gen/internal/cases"
	"golang.org/x/tools/txtar"
)

func TestTheExampleIsTheCaseThatReadsIt(t *testing.T) {
	given, err := os.ReadFile("../../examples/comments.lex")

	if err != nil {
		t.Fatal(err)
	}

	archive, err := txtar.ParseFile("testdata/cases.txtar")

	if err != nil {
		t.Fatal(err)
	}

	at := slices.IndexFunc(archive.Files, func(f txtar.File) bool { return f.Name == "comments" })

	if at < 0 || string(archive.Files[at].Data) != string(given) {
		t.Errorf("the comments case is not examples/comments.lex:\n%s", given)
	}
}

func TestTheExampleReadsNestedComments(t *testing.T) {
	got, err := lextest.Read(t, cases.Lexers["comments"], "a /* b /* c */ d */ e\n")

	if want := []string{"word(a)", "open(/*)", "open(/*)", "close(*/)", "close(*/)", "word(e)", "newline(\n)"}; err != nil || !reflect.DeepEqual(got, want) {
		t.Errorf("read %q, %v; want %q", got, err, want)
	}
}
