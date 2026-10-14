package fa_test

import (
	"math"
	"reflect"
	"testing"
	"unicode"

	"github.com/cptaffe/automaton/fa"
)

type ranges = []fa.Range[int]

func TestUnionMergesRangesThatOverlapOrTouch(t *testing.T) {
	for _, c := range []struct {
		name  string
		given fa.Set[int]
		when  fa.Set[int]
		then  ranges
	}{
		{"overlapping", fa.Span(1, 5), fa.Span(3, 9), ranges{{1, 9}}},
		{"sharing an endpoint", fa.Span(1, 5), fa.Span(5, 9), ranges{{1, 9}}},
		{"adjacent", fa.Span(1, 4), fa.Span(5, 9), ranges{{1, 9}}},
		{"apart", fa.Span(1, 3), fa.Span(5, 9), ranges{{1, 3}, {5, 9}}},
		{"the later one first", fa.Span(5, 9), fa.Span(1, 5), ranges{{1, 9}}},
		{"one inside the other", fa.Span(1, 9), fa.Span(3, 4), ranges{{1, 9}}},
		{"filling a gap", fa.Of(fa.Range[int]{1, 2}, fa.Range[int]{6, 7}), fa.Span(3, 5), ranges{{1, 7}}},
		{"a backward range", fa.Span(1, 5), fa.Span(9, 1), ranges{{1, 5}}},
		{"onto nothing", fa.Set[int]{}, fa.One(3), ranges{{3, 3}}},
	} {
		t.Run(c.name, func(t *testing.T) {
			got := c.given.Union(c.when).Ranges()

			if !reflect.DeepEqual(got, c.then) {
				t.Errorf("%v with %v is %v; want %v", c.given.Ranges(), c.when.Ranges(), got, c.then)
			}
		})
	}
}

func TestIntersectKeepsWhatBothSetsHold(t *testing.T) {
	several := fa.Of(fa.Range[int]{1, 2}, fa.Range[int]{4, 5}, fa.Range[int]{7, 8})

	for _, c := range []struct {
		name  string
		given fa.Set[int]
		when  fa.Set[int]
		then  ranges
	}{
		{"overlapping", fa.Span(1, 13), fa.Span(8, 26), ranges{{8, 13}}},
		{"apart", fa.Span(1, 5), fa.Span(7, 9), nil},
		{"several ranges inside a wider one", several, fa.Span(0, 10), ranges{{1, 2}, {4, 5}, {7, 8}}},
		{"a wider range around several", fa.Span(0, 10), several, ranges{{1, 2}, {4, 5}, {7, 8}}},
		{"interleaved", fa.Of(fa.Range[int]{1, 4}, fa.Range[int]{8, 12}), fa.Of(fa.Range[int]{3, 9}, fa.Range[int]{11, 20}), ranges{{3, 4}, {8, 9}, {11, 12}}},
	} {
		t.Run(c.name, func(t *testing.T) {
			got := c.given.Intersect(c.when).Ranges()

			if !reflect.DeepEqual(got, c.then) {
				t.Errorf("%v and %v share %v; want %v", c.given.Ranges(), c.when.Ranges(), got, c.then)
			}
		})
	}
}

func TestWithoutRemovesWhatTheOtherSetHolds(t *testing.T) {
	for _, c := range []struct {
		name  string
		given fa.Set[int]
		when  fa.Set[int]
		then  ranges
	}{
		{"overlapping", fa.Span(1, 13), fa.Span(8, 26), ranges{{1, 7}}},
		{"holes", fa.Span(0, 10), fa.Of(fa.Range[int]{2, 3}, fa.Range[int]{6, 6}), ranges{{0, 1}, {4, 5}, {7, 10}}},
		{"all of it", fa.Span(1, 5), fa.Span(0, 9), nil},
		{"apart", fa.Span(1, 5), fa.Span(7, 9), ranges{{1, 5}}},
	} {
		t.Run(c.name, func(t *testing.T) {
			got := c.given.Without(c.when).Ranges()

			if !reflect.DeepEqual(got, c.then) {
				t.Errorf("%v without %v is %v; want %v", c.given.Ranges(), c.when.Ranges(), got, c.then)
			}
		})
	}
}

func TestNegateIsAnInverse(t *testing.T) {
	s := fa.Text("abc").Union(fa.Span('0', '9'))

	for _, r := range []rune{'a', 'b', 'c', '0', '5', '9', 'd', 'A', 0, 0x10FFFF, '\n'} {
		if s.Has(r) == s.Negate().Has(r) {
			t.Errorf("%q is in both a set and its negation", r)
		}
	}

	if got := s.Negate().Negate(); !got.Equal(s) {
		t.Errorf("negating twice gives %s; want %s", got, s)
	}
}

func TestCategoryKeepsEveryRuneOfAStride(t *testing.T) {
	given := &unicode.RangeTable{
		R16: []unicode.Range16{{Lo: 'a', Hi: 'g', Stride: 3}},
		R32: []unicode.Range32{{Lo: 0x10000, Hi: 0x10008, Stride: 4}, {Lo: 0x20000, Hi: 0x20002, Stride: 1}},
	}

	got := fa.Category(given).Ranges()

	then := []fa.Range[rune]{{'a', 'a'}, {'d', 'd'}, {'g', 'g'}, {0x10000, 0x10000}, {0x10004, 0x10004}, {0x10008, 0x10008}, {0x20000, 0x20002}}

	if !reflect.DeepEqual(got, then) {
		t.Errorf("the category is %v; want %v", got, then)
	}
}

func TestCategoryHoldsTheEndsOfUnicodeStrides(t *testing.T) {
	for _, c := range []struct {
		name  string
		given *unicode.RangeTable
		when  rune
		then  bool
	}{
		{"the first upper case letter of a stride", unicode.Upper, 'Ā', true},
		{"the last upper case letter of a stride", unicode.Upper, 'Ķ', true},
		{"a lower case letter between upper ones", unicode.Upper, 'ķ', false},
		{"the last lower case letter of a stride", unicode.Lower, 'ķ', true},
		{"the end of a stride of 42", unicode.Lower, 'ß', true},
		{"a rune inside a stride of 42", unicode.Lower, 'Ê', false},
	} {
		t.Run(c.name, func(t *testing.T) {
			if got := fa.Category(c.given).Has(c.when); got != c.then {
				t.Errorf("%U is in the category: %t; want %t", c.when, got, c.then)
			}
		})
	}
}

func TestThePartitionIsCoarsest(t *testing.T) {
	letters, digits := fa.Span('a', 'z'), fa.Span('0', '9')
	c := fa.Partition([]fa.Set[rune]{letters, digits})

	if c.Len() != 2 {
		t.Fatalf("letters and digits make %d parts; want 2", c.Len())
	}

	if c.Of('a') != c.Of('q') || c.Of('a') != c.Of('z') {
		t.Error("two letters landed in different parts, and no set tells them apart")
	}

	if c.Of('a') == c.Of('5') {
		t.Error("a letter and a digit landed in one part")
	}

	if c.Of('!') != fa.NoClass {
		t.Error("a rune in no set got a part")
	}
}

func TestPartitionCutsSetsWhereTheyOverlap(t *testing.T) {
	for _, c := range []struct {
		name  string
		given []fa.Set[rune]
		then  []string
	}{
		{"apart", []fa.Set[rune]{fa.Span('a', 'z'), fa.Span('0', '9')}, []string{"0-9", "a-z"}},
		{"overlapping", []fa.Set[rune]{fa.Span('a', 'm'), fa.Span('h', 'z')}, []string{"a-g", "h-m", "n-z"}},
		{"one inside another", []fa.Set[rune]{fa.Span('a', 'z'), fa.One('q')}, []string{"a-pr-z", "q"}},
		{"two ranges that nothing splits", []fa.Set[rune]{fa.Of(fa.Range[rune]{'a', 'c'}, fa.Range[rune]{'x', 'z'}), fa.Span('0', '9')}, []string{"0-9", "a-cx-z"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			p := fa.Partition(c.given)
			got := make([]string, 0, p.Len())

			for at := fa.Class(1); int(at) <= p.Len(); at++ {
				got = append(got, p.Part(at).String())
			}

			if !reflect.DeepEqual(got, c.then) {
				t.Errorf("the parts are %q; want %q", got, c.then)
			}
		})
	}
}

func TestEverySetIsAUnionOfParts(t *testing.T) {
	sets := []fa.Set[rune]{
		fa.Span('a', 'm'),
		fa.Span('h', 'z'),
		fa.Text("*#-"),
		fa.Span('0', '9'),
	}
	c := fa.Partition(sets)

	for _, s := range sets {
		var built fa.Set[rune]

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
	letters := fa.Category(unicode.L)

	if n := letters.Size(); n < 100_000 {
		t.Fatalf("unicode letters are %d runes; the table is not loaded", n)
	}

	c := fa.Partition([]fa.Set[rune]{letters, fa.Category(unicode.Nd)})

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

func TestARuneSetPrintsAsALabel(t *testing.T) {
	for _, c := range []struct {
		name  string
		given fa.Set[rune]
		then  string
	}{
		{"a range", fa.Span('a', 'e'), "a-e"},
		{"two runes side by side", fa.Span('a', 'b'), "ab"},
		{"runes and ranges run together", fa.Text("*#0123456789"), "#*0-9"},
		{"a lone rune before a pair", fa.Text("acd"), "acd"},
		{"nothing", fa.Set[rune]{}, "nothing"},
	} {
		t.Run(c.name, func(t *testing.T) {
			if got := c.given.String(); got != c.then {
				t.Errorf("%v prints as %q; want %q", c.given.Ranges(), got, c.then)
			}
		})
	}
}

func TestALabelShowsWhatStandsAlone(t *testing.T) {
	for _, c := range []struct {
		name  string
		given rune
		then  string
	}{
		{"a latin letter", 'a', "a"},
		{"a han character", '漢', "漢"},
		{"a greek letter", 'α', "α"},
		{"an eszett", 'ß', "ß"},
		{"a combining accent", '́', "\\u0301"},
		{"a zero width joiner", '‍', "\\u200d"},
		{"the last rune of the basic plane", '￿', "\\uffff"},
		{"a format rune past the basic plane", '\U000e0001', "\\u0e0001"},
		{"a newline", '\n', "\\n"},
		{"a tab", '\t', "\\t"},
		{"a space", ' ', "␠"},
	} {
		t.Run(c.name, func(t *testing.T) {
			if got := fa.One(c.given).String(); got != c.then {
				t.Errorf("%U shows as %q; want %q", c.given, got, c.then)
			}
		})
	}
}

func TestANumberSetPrintsItsRanges(t *testing.T) {
	given := fa.Of(fa.Range[int]{1, 1}, fa.Range[int]{3, 4}, fa.Range[int]{6, 9})

	got := given.String()

	if then := "1 3..4 6..9"; got != then {
		t.Errorf("the set prints as %q; want %q", got, then)
	}
}

func TestASetReachesTheTopOfItsType(t *testing.T) {
	top := fa.Span[byte](250, 255)
	if got := top.Union(fa.One[byte](249)); got.String() != "249..255" {
		t.Errorf("249 and 250..255 are %s", got)
	}
	if got := top.Negate(); !got.Equal(fa.Span[byte](0, 249)) {
		t.Errorf("the rest of the bytes are %s", got)
	}
	if got := fa.All[int8]().Size(); got != 256 {
		t.Errorf("an int8 holds %d values", got)
	}
	if got := fa.All[int64]().Without(fa.One[int64](math.MaxInt64)); got.Has(math.MaxInt64) || !got.Has(math.MinInt64) {
		t.Errorf("all int64 but the top is %s", got)
	}

	c := fa.Partition([]fa.Set[byte]{top, fa.Span[byte](0, 255)})
	if c.Len() != 2 || c.Of(255) == fa.NoClass || c.Of(255) != c.Of(250) || c.Of(0) == c.Of(255) {
		t.Errorf("the classes of 0..255 split at 250 are %d, and 255 is in %d", c.Len(), c.Of(255))
	}
}
