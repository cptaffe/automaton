package draw

import (
	"fmt"

	"github.com/cptaffe/automaton"
)

const wide = 24

func label(s automaton.Set) string {
	if s.Empty() {
		return "nothing"
	}
	direct, whole := spell(s)
	not := s.Negate()
	if not.Empty() {
		return direct
	}
	left, all := spell(not)
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

func spelled(s automaton.Set) string {
	out, _ := spell(s)
	return out
}

func spell(s automaton.Set) (string, bool) {
	if whole := s.String(); len(whole) <= wide {
		return whole, true
	}
	rs := s.Ranges()
	at := len(rs)
	for at > 1 && len(automaton.Of(rs[:at]...).String()) > wide {
		at--
	}
	n := 0
	for _, x := range rs[at:] {
		n += int(x.Hi-x.Lo) + 1
	}
	return automaton.Of(rs[:at]...).String() + fmt.Sprintf(" and %d more", n), false
}
