package fa

import (
	"math/rand/v2"
	"testing"
)

func sparse(n, k int) ([][]int, []string) {
	r := rand.New(rand.NewPCG(1, 2))
	next := make([][]int, n)
	sig := make([]string, n)
	for s := range n {
		next[s] = make([]int, k)
		if s == 0 {
			continue
		}
		for range 3 {
			next[s][r.IntN(k)] = r.IntN(n)
		}
		sig[s] = string(rune('a' + r.IntN(4)))
	}
	return next, sig
}

func BenchmarkEquivalent(b *testing.B) {
	next, sig := sparse(2000, 30)
	for b.Loop() {
		Equivalent(next, sig)
	}
}
