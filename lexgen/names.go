package lexgen

import (
	"sort"
	"strings"
	"unicode"

	"github.com/cptaffe/automaton"
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

var unicodeClasses = func() map[string]automaton.Set {
	out := map[string]automaton.Set{}

	for _, from := range []map[string]*unicode.RangeTable{
		unicode.Categories, unicode.Scripts, unicode.Properties,
	} {
		for name, t := range from {
			out[kebab(name)] = automaton.Category(t)
		}
	}

	return out
}()

func unicodeClass(name string) (automaton.Set, bool) {
	rest, ok := strings.CutPrefix(name, namespace)

	if !ok {
		return automaton.Set{}, false
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

func unicodeNames() []string {
	out := make([]string, 0, len(unicode.Categories)+len(unicode.Properties))

	for n := range unicode.Categories {
		out = append(out, namespace+kebab(n))
	}

	for n := range unicode.Properties {
		out = append(out, namespace+kebab(n))
	}

	sort.Strings(out)

	return out
}
