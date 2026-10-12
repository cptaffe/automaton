package automaton

import (
	"fmt"
	"sort"
	"strings"
	"unicode"
	"unsafe"
)

type Symbol interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 | ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr
}

func top[S Symbol]() S {
	var z S
	if ^z > z {
		return ^z
	}
	return S(^uint64(0) >> (65 - 8*unsafe.Sizeof(z)))
}

func bottom[S Symbol]() S {
	var z S
	if ^z > z {
		return 0
	}
	return -top[S]() - 1
}

type Range[S Symbol] struct {
	Lo, Hi S
}

type Set[S Symbol] struct {
	r []Range[S]
}

func Of[S Symbol](rs ...Range[S]) Set[S] {
	out := make([]Range[S], 0, len(rs))

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

	var norm []Range[S]

	for _, x := range out {
		if n := len(norm); n > 0 && (x.Lo <= norm[n-1].Hi || norm[n-1].Hi != top[S]() && x.Lo == norm[n-1].Hi+1) {
			norm[n-1].Hi = max(norm[n-1].Hi, x.Hi)

			continue
		}

		norm = append(norm, x)
	}

	return Set[S]{r: norm}
}

func One[S Symbol](x S) Set[S] { return Of(Range[S]{x, x}) }

func Span[S Symbol](lo, hi S) Set[S] { return Of(Range[S]{lo, hi}) }

func All[S Symbol]() Set[S] { return Span(bottom[S](), top[S]()) }

func Text(s string) Set[rune] {
	var rs []Range[rune]

	for _, r := range s {
		rs = append(rs, Range[rune]{r, r})
	}

	return Of(rs...)
}

func Category(t *unicode.RangeTable) Set[rune] {
	var rs []Range[rune]

	add := func(lo, hi, stride rune) {
		if stride == 1 {
			rs = append(rs, Range[rune]{lo, hi})

			return
		}

		for r := lo; r <= hi; r += stride {
			rs = append(rs, Range[rune]{r, r})
		}
	}

	for _, x := range t.R16 {
		add(rune(x.Lo), rune(x.Hi), rune(x.Stride))
	}

	for _, x := range t.R32 {
		add(rune(x.Lo), rune(x.Hi), rune(x.Stride))
	}

	return Of(rs...)
}

func (s Set[S]) Ranges() []Range[S] { return append([]Range[S](nil), s.r...) }

func (s Set[S]) Empty() bool { return len(s.r) == 0 }

func (s Set[S]) Has(x S) bool {
	at := sort.Search(len(s.r), func(i int) bool { return s.r[i].Hi >= x })

	return at < len(s.r) && s.r[at].Lo <= x
}

func (s Set[S]) Size() uint64 {
	var n uint64

	for _, x := range s.r {
		n += uint64(x.Hi) - uint64(x.Lo) + 1
	}

	return n
}

func (s Set[S]) Union(t Set[S]) Set[S] {
	return Of(append(append([]Range[S](nil), s.r...), t.r...)...)
}

func (s Set[S]) Intersect(t Set[S]) Set[S] {
	var out []Range[S]
	i, j := 0, 0

	for i < len(s.r) && j < len(t.r) {
		lo, hi := max(s.r[i].Lo, t.r[j].Lo), min(s.r[i].Hi, t.r[j].Hi)

		if lo <= hi {
			out = append(out, Range[S]{lo, hi})
		}

		if s.r[i].Hi < t.r[j].Hi {
			i++

			continue
		}

		j++
	}

	return Of(out...)
}

func (s Set[S]) Without(t Set[S]) Set[S] { return s.Intersect(t.Negate()) }

func (s Set[S]) Negate() Set[S] {
	var out []Range[S]
	at, open := bottom[S](), true

	for _, x := range s.r {
		if x.Lo > at {
			out = append(out, Range[S]{at, x.Lo - 1})
		}

		if x.Hi == top[S]() {
			open = false

			break
		}

		at = x.Hi + 1
	}

	if open {
		out = append(out, Range[S]{at, top[S]()})
	}

	return Of(out...)
}

func (s Set[S]) Equal(t Set[S]) bool {
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

func (s Set[S]) String() string {
	if s.Empty() {
		return "nothing"
	}

	var b strings.Builder

	for i, x := range s.r {
		_, runes := any(x.Lo).(rune)

		if i > 0 && !runes {
			b.WriteByte(' ')
		}

		switch {
		case x.Lo == x.Hi:
			b.WriteString(show(x.Lo))
		case runes && x.Hi == x.Lo+1:
			b.WriteString(show(x.Lo) + show(x.Hi))
		case runes:
			b.WriteString(show(x.Lo) + "-" + show(x.Hi))
		default:
			b.WriteString(show(x.Lo) + ".." + show(x.Hi))
		}
	}

	return b.String()
}

func show[S Symbol](x S) string {
	r, ok := any(x).(rune)

	if !ok {
		return fmt.Sprint(x)
	}

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
