package fa_test

import (
	"regexp"
	"strings"
	"testing"

	"github.com/cptaffe/automaton/fa"
	"github.com/cptaffe/automaton/internal/regextest"
)

func accepts(d *fa.DFA[rune], w string) bool {
	s := d.Start()
	for _, r := range w {
		s = d.Step(s, r)
	}
	return d.Accept(s) != fa.NoMatch
}

func FuzzBuildAgreesWithRegexp(f *testing.F) {
	for _, seed := range [][]byte{{0, 3}, {1, 0, 1, 0, 2}, {3, 2, 0, 1, 0, 4}, {4, 1, 0, 5, 5, 0, 2}} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		p := regextest.Decode(data)
		e, src := p.FA, p.Regexp
		re := regexp.MustCompile("^(?:" + src + ")$")
		rules := []fa.Rule[rune]{{Expr: e, Name: src}}

		d, err := fa.BuildWith(rules, fa.Split(fa.Sets(rules)), 1000)

		if err != nil {
			if strings.Contains(err.Error(), "matches nothing at all") {
				return
			}
			t.Fatalf("%s: %v", src, err)
		}
		for _, w := range regextest.Words(4) {
			if got, want := accepts(d, w), re.MatchString(w); got != want {
				t.Fatalf("%s on %q: fa says %v, regexp %v", src, w, got, want)
			}
			if got, want := accepts(d.Minimise(), w), re.MatchString(w); got != want {
				t.Fatalf("%s on %q minimised: fa says %v, regexp %v", src, w, got, want)
			}
		}
	})
}
