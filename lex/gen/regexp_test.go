package gen_test

import (
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/cptaffe/automaton/internal/lextest"
	"github.com/cptaffe/automaton/internal/regextest"
)

func longest(re *regexp.Regexp, text string) []string {
	var out []string
	for at := 0; at < len(text); {
		if m := re.FindStringIndex(text[at:]); m != nil && m[1] > 0 {
			out = append(out, "x("+text[at:at+m[1]]+")")
			at += m[1]
			continue
		}
		at++
	}
	return out
}

func FuzzLexAgreesWithRegexp(f *testing.F) {
	f.Add([]byte{4, 0, 1}, "aab")
	f.Add([]byte{1, 0, 3, 3, 0, 5}, "abcab")
	f.Add([]byte{2, 0, 1, 0, 2}, "cabbage")
	f.Fuzz(func(t *testing.T, pattern []byte, text string) {
		text = strings.Map(func(r rune) rune {
			if strings.ContainsRune(regextest.Letters, r) {
				return r
			}
			return -1
		}, text)
		if len(text) > 12 {
			text = text[:12]
		}
		p := regextest.Decode(pattern)
		src := "start {\n  " + p.Lex + "  emit x\n  any  ignore\n}\n"
		if lextest.Error(src) != nil {
			return
		}
		re := regexp.MustCompile("^(?:" + p.Regexp + ")")
		re.Longest()

		got, err := lextest.Read(t, lextest.Compile(t, src), text)

		if want := longest(re, text); err != nil || !reflect.DeepEqual(got, want) && len(got)+len(want) > 0 {
			t.Fatalf("%s on %q: lex read %q, %v; regexp %q", p.Lex, text, got, err, want)
		}
	})
}
