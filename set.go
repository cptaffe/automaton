package automaton

import (
	"fmt"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

type Range struct {
	Lo, Hi rune
}

type Set struct {
	r []Range
}

func Of(rs ...Range) Set {
	out := make([]Range, 0, len(rs))

	for _, x := range rs {
		if x.Lo <= x.Hi {
			out = append(out, x)
		}
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].Lo != out[j].Lo {
			return out[i].Lo < out[j].Lo
		}

		return out[i].Hi < out[j].Hi
	})

	var norm []Range

	for _, x := range out {
		if n := len(norm); n > 0 && x.Lo <= norm[n-1].Hi+1 {
			if x.Hi > norm[n-1].Hi {
				norm[n-1].Hi = x.Hi
			}

			continue
		}

		norm = append(norm, x)
	}

	return Set{r: norm}
}

func One(r rune) Set { return Of(Range{r, r}) }

func Span(lo, hi rune) Set { return Of(Range{lo, hi}) }

func Text(s string) Set {
	var rs []Range

	for _, r := range s {
		rs = append(rs, Range{r, r})
	}

	return Of(rs...)
}

func Category(t *unicode.RangeTable) Set {
	var rs []Range

	for _, x := range t.R16 {
		if x.Stride != 1 {
			for r := rune(x.Lo); r <= rune(x.Hi); r += rune(x.Stride) {
				rs = append(rs, Range{r, r})
			}

			continue
		}

		rs = append(rs, Range{rune(x.Lo), rune(x.Hi)})
	}

	for _, x := range t.R32 {
		if x.Stride != 1 {
			for r := rune(x.Lo); r <= rune(x.Hi); r += rune(x.Stride) {
				rs = append(rs, Range{r, r})
			}

			continue
		}

		rs = append(rs, Range{rune(x.Lo), rune(x.Hi)})
	}

	return Of(rs...)
}

func (s Set) Ranges() []Range { return s.r }

func (s Set) Empty() bool { return len(s.r) == 0 }

func (s Set) Has(r rune) bool {
	at := sort.Search(len(s.r), func(i int) bool { return s.r[i].Hi >= r })

	return at < len(s.r) && s.r[at].Lo <= r
}

func (s Set) Size() int {
	n := 0

	for _, x := range s.r {
		n += int(x.Hi-x.Lo) + 1
	}

	return n
}

func (s Set) Union(t Set) Set {
	return Of(append(append([]Range(nil), s.r...), t.r...)...)
}

func (s Set) Intersect(t Set) Set {
	var out []Range
	i, j := 0, 0

	for i < len(s.r) && j < len(t.r) {
		lo, hi := max(s.r[i].Lo, t.r[j].Lo), min(s.r[i].Hi, t.r[j].Hi)

		if lo <= hi {
			out = append(out, Range{lo, hi})
		}

		if s.r[i].Hi < t.r[j].Hi {
			i++

			continue
		}

		j++
	}

	return Of(out...)
}

func (s Set) Without(t Set) Set { return s.Intersect(t.Negate()) }

func (s Set) Negate() Set {
	var out []Range
	at := rune(0)

	for _, x := range s.r {
		if x.Lo > at {
			out = append(out, Range{at, x.Lo - 1})
		}

		if x.Hi+1 > at {
			at = x.Hi + 1
		}
	}

	if at <= utf8.MaxRune {
		out = append(out, Range{at, utf8.MaxRune})
	}

	return Of(out...)
}

func (s Set) Equal(t Set) bool {
	if len(s.r) != len(t.r) {
		return false
	}

	for i := range s.r {
		if s.r[i] != t.r[i] {
			return false
		}
	}

	return true
}

func (s Set) String() string {
	if s.Empty() {
		return "nothing"
	}

	var b strings.Builder

	for _, x := range s.r {
		switch {
		case x.Lo == x.Hi:
			b.WriteString(show(x.Lo))
		case x.Hi == x.Lo+1:
			b.WriteString(show(x.Lo) + show(x.Hi))
		default:
			b.WriteString(show(x.Lo) + "-" + show(x.Hi))
		}
	}

	return b.String()
}

func (s Set) Label() string {
	if s.Empty() {
		return "nothing"
	}

	direct, whole := s.spelled()

	not := s.Negate()

	if not.Empty() {
		return direct
	}

	left, all := not.spelled()
	away := "any - [" + left + "]"

	switch {
	case whole && !all:
		return direct
	case all && !whole:
		return away
	case len(away) < len(direct):
		return away
	}

	return direct
}

func (s Set) Spelled() string {
	out, _ := s.spelled()

	return out
}

func (s Set) spelled() (string, bool) {

	const wide = 24

	if whole := s.String(); len(whole) <= wide {
		return whole, true
	}

	at := len(s.r)

	for at > 1 && len(Set{r: s.r[:at]}.String()) > wide {
		at--
	}

	n := 0

	for _, x := range s.r[at:] {
		n += int(x.Hi-x.Lo) + 1
	}

	return Set{r: s.r[:at]}.String() + fmt.Sprintf(" and %d more", n), false
}

func show(r rune) string {
	switch r {
	case '\n':
		return `\n`
	case '\t':
		return `\t`
	case ' ':
		return `␠`
	}

	if !unicode.IsPrint(r) || unicode.In(r, unicode.M, unicode.Cf) {
		return "\\u" + hex(r)
	}

	return string(r)
}

func hex(r rune) string {
	const digits = "0123456789abcdef"
	n := 4

	if r > 0xffff {
		n = 6
	}

	out := []byte{}

	for i := n - 1; i >= 0; i-- {
		out = append(out, digits[(r>>(4*i))&0xf])
	}

	return string(out)
}
