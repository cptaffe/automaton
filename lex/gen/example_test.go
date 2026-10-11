package gen_test

import (
	"os"
	"reflect"
	"testing"
)

func TestTheExampleLexes(t *testing.T) {
	src, err := os.ReadFile("../../examples/comments.lex")
	if err != nil {
		t.Fatal(err)
	}
	got := kinds(t, build(t, string(src)), "a /* b /* c */ d */ e\n")
	want := []string{"word(a)", "open(/*)", "open(/*)", "close(*/)", "close(*/)", "word(e)", "newline(\n)"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}
