package draw

import (
	"fmt"
	"unicode/utf8"

	"github.com/cptaffe/automaton/fa"
)

const wide = 24

func label(s fa.Set[rune]) string {
	if s.Empty() {
		return "nothing"
	}
	direct, whole := spell(s)
	not := fa.Span[rune](0, utf8.MaxRune).Without(s)
	if not.Empty() {
		return direct
	}
	left, all := spell(not)
	away := "any \\ [" + left + "]"
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

func spelled(s fa.Set[rune]) string {
	out, _ := spell(s)
	return out
}

func spell(s fa.Set[rune]) (string, bool) {
	if whole := s.String(); len(whole) <= wide {
		return whole, true
	}
	rs := s.Ranges()
	at := len(rs)
	for at > 1 && len(fa.Of(rs[:at]...).String()) > wide {
		at--
	}
	n := 0
	for _, x := range rs[at:] {
		n += int(x.Hi-x.Lo) + 1
	}
	return fa.Of(rs[:at]...).String() + fmt.Sprintf(" and %d more", n), false
}
