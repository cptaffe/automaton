package fa_test

import (
	"testing"
	"unicode"

	"github.com/cptaffe/automaton/fa"
)

func BenchmarkPartitionUnicode(b *testing.B) {
	sets := []fa.Set[rune]{
		fa.Category(unicode.L),
		fa.Category(unicode.Nd),
		fa.Category(unicode.Space),
		fa.Category(unicode.P),
		fa.Text("*#-+"),
	}

	for b.Loop() {
		fa.Partition(sets)
	}
}
