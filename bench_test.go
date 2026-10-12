package automaton_test

import (
	"testing"
	"unicode"

	"github.com/cptaffe/automaton"
)

func BenchmarkPartitionUnicode(b *testing.B) {
	sets := []automaton.Set[rune]{
		automaton.Category(unicode.L),
		automaton.Category(unicode.Nd),
		automaton.Category(unicode.Space),
		automaton.Category(unicode.P),
		automaton.Text("*#-+"),
	}

	for b.Loop() {
		automaton.Partition(sets)
	}
}
