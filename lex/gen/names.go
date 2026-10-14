package gen

import (
	"strings"
	"unicode"

	"github.com/cptaffe/automaton/fa"
)

const namespace = "unicode."

func kebab(name string) string {
	var b strings.Builder

	for _, r := range name {
		switch r {
		case '_', ' ', '\t':
			b.WriteByte('-')
		default:
			b.WriteRune(unicode.ToLower(r))
		}
	}

	return b.String()
}

var unicodeClasses = func() map[string]fa.Set[rune] {
	out := map[string]fa.Set[rune]{}

	for _, from := range []map[string]*unicode.RangeTable{
		unicode.Categories, unicode.Scripts, unicode.Properties,
	} {
		for name, t := range from {
			out[kebab(name)] = fa.Category(t)
		}
	}

	return out
}()

func unicodeClass(name string) (fa.Set[rune], bool) {
	rest, ok := strings.CutPrefix(name, namespace)

	if !ok {
		return fa.Set[rune]{}, false
	}

	s, found := unicodeClasses[rest]

	return s, found
}

func spelling(name string) string {
	rest, ok := strings.CutPrefix(name, namespace)

	if !ok {
		rest = name
	}

	at := kebab(rest)

	if _, found := unicodeClasses[at]; !found {
		return ""
	}

	if also := namespace + at; also != name {
		return also
	}

	return ""
}
