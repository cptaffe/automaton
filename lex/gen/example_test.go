package gen_test

import (
	"os"
	"reflect"
	"testing"

	"github.com/cptaffe/automaton/internal/lextest"
)

func TestTheExampleReadsNestedComments(t *testing.T) {
	given, err := os.ReadFile("../../examples/comments.lex")

	if err != nil {
		t.Fatal(err)
	}

	got, err := lextest.Read(t, lextest.Compile(t, string(given)), "a /* b /* c */ d */ e\n")

	if want := []string{"word(a)", "open(/*)", "open(/*)", "close(*/)", "close(*/)", "word(e)", "newline(\n)"}; err != nil || !reflect.DeepEqual(got, want) {
		t.Errorf("read %q, %v; want %q", got, err, want)
	}
}
