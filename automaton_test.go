package automaton_test

import (
	"testing"
	"unicode"

	"github.com/cptaffe/automaton"
)

func TestASetNormalises(t *testing.T) {
	for _, c := range []struct {
		of   automaton.Set[rune]
		want string
	}{
		{automaton.Of(automaton.Range[rune]{'a', 'c'}, automaton.Range[rune]{'b', 'e'}), "a-e"},
		{automaton.Of(automaton.Range[rune]{'a', 'b'}, automaton.Range[rune]{'c', 'd'}), "a-d"},
		{automaton.Of(automaton.Range[rune]{'c', 'd'}, automaton.Range[rune]{'a', 'a'}), "acd"},
		{automaton.Text("*#0123456789"), "#*0-9"},
		{automaton.Of(automaton.Range[rune]{'z', 'a'}), "nothing"},
	} {
		if got := c.of.String(); got != c.want {
			t.Errorf("%v is %q; want %q", c.of.Ranges(), got, c.want)
		}
	}
}

func TestNegateIsAnInverse(t *testing.T) {
	s := automaton.Text("abc").Union(automaton.Span('0', '9'))

	for _, r := range []rune{'a', 'b', 'c', '0', '5', '9', 'd', 'A', 0, 0x10FFFF, '\n'} {
		if s.Has(r) == s.Negate().Has(r) {
			t.Errorf("%q is in both a set and its negation", r)
		}
	}

	if got := s.Negate().Negate(); !got.Equal(s) {
		t.Errorf("negating twice gives %s; want %s", got, s)
	}
}

func TestIntersectAndWithout(t *testing.T) {
	a, b := automaton.Span('a', 'm'), automaton.Span('h', 'z')

	if got := a.Intersect(b).String(); got != "h-m" {
		t.Errorf("the overlap is %q", got)
	}

	if got := a.Without(b).String(); got != "a-g" {
		t.Errorf("what is left is %q", got)
	}

	if got := a.Intersect(automaton.Span('0', '9')); !got.Empty() {
		t.Errorf("letters and digits share %s", got)
	}
}

func TestThePartitionIsCoarsest(t *testing.T) {
	letters, digits := automaton.Span('a', 'z'), automaton.Span('0', '9')
	c := automaton.Partition([]automaton.Set[rune]{letters, digits})

	if c.Len() != 2 {
		t.Fatalf("letters and digits make %d parts; want 2", c.Len())
	}

	if c.Of('a') != c.Of('q') || c.Of('a') != c.Of('z') {
		t.Error("two letters landed in different parts, and no set tells them apart")
	}

	if c.Of('a') == c.Of('5') {
		t.Error("a letter and a digit landed in one part")
	}

	if c.Of('!') != automaton.NoClass {
		t.Error("a rune in no set got a part")
	}
}

func TestAnOverlapSplits(t *testing.T) {
	c := automaton.Partition([]automaton.Set[rune]{automaton.Span('a', 'm'), automaton.Span('h', 'z')})

	if c.Len() != 3 {
		t.Fatalf("a-m and h-z make %d parts; want 3 (a-g, h-m, n-z)", c.Len())
	}

	if c.Of('a') == c.Of('h') || c.Of('h') == c.Of('n') || c.Of('a') == c.Of('n') {
		t.Error("the three parts are not distinct")
	}

	if c.Of('a') != c.Of('g') || c.Of('h') != c.Of('m') || c.Of('n') != c.Of('z') {
		t.Error("a part is split where nothing tells it apart")
	}
}

func TestEverySetIsAUnionOfParts(t *testing.T) {
	sets := []automaton.Set[rune]{
		automaton.Span('a', 'm'),
		automaton.Span('h', 'z'),
		automaton.Text("*#-"),
		automaton.Span('0', '9'),
	}
	c := automaton.Partition(sets)

	for _, s := range sets {
		var built automaton.Set[rune]

		for _, at := range c.Covers(s) {
			part := c.Part(at)

			if got := part.Without(s); !got.Empty() {
				t.Errorf("part %s is not wholly inside %s: %s is outside", part, s, got)
			}

			built = built.Union(part)
		}

		if !built.Equal(s) {
			t.Errorf("%s is built from its parts as %s", s, built)
		}
	}
}

func TestUnicodeStaysTwoColumns(t *testing.T) {
	letters := automaton.Category(unicode.L)

	if n := letters.Size(); n < 100_000 {
		t.Fatalf("unicode letters are %d runes; the table is not loaded", n)
	}

	c := automaton.Partition([]automaton.Set[rune]{letters, automaton.Category(unicode.Nd)})

	if c.Len() != 2 {
		t.Errorf("letters and decimal digits make %d parts; want 2", c.Len())
	}

	if c.Of('a') != c.Of('ß') || c.Of('a') != c.Of('漢') {
		t.Error("a latin letter, an eszett and a han character are not one class")
	}

	if c.Of('a') == c.Of('5') {
		t.Error("a letter and a digit are one class")
	}
}

func TestALabelShowsWhatStandsAlone(t *testing.T) {
	for _, c := range []struct {
		in   rune
		want string
	}{
		{'a', "a"},
		{'\u6f22', "\u6f22"},
		{'\u03b1', "\u03b1"},
		{'\u00df', "\u00df"},
		{'\u0301', "\\u0301"},
		{'\u200d', "\\u200d"},
		{'\n', "\\n"},
		{' ', "\u2420"},
	} {
		if got := automaton.One(c.in).String(); got != c.want {
			t.Errorf("%U shows as %q; want %q", c.in, got, c.want)
		}
	}
}
