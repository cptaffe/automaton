package automaton_test

import (
	"math"
	"testing"

	"github.com/cptaffe/automaton"
)

func TestASetReachesTheTopOfItsType(t *testing.T) {
	top := automaton.Span[byte](250, 255)
	if got := top.Union(automaton.One[byte](249)); got.String() != "249..255" {
		t.Errorf("249 and 250..255 are %s", got)
	}
	if got := top.Negate(); !got.Equal(automaton.Span[byte](0, 249)) {
		t.Errorf("the rest of the bytes are %s", got)
	}
	if got := automaton.All[int8]().Size(); got != 256 {
		t.Errorf("an int8 holds %d values", got)
	}
	if got := automaton.All[int64]().Without(automaton.One[int64](math.MaxInt64)); got.Has(math.MaxInt64) || !got.Has(math.MinInt64) {
		t.Errorf("all int64 but the top is %s", got)
	}

	c := automaton.Partition([]automaton.Set[byte]{top, automaton.Span[byte](0, 255)})
	if c.Len() != 2 || c.Of(255) == automaton.NoClass || c.Of(255) != c.Of(250) || c.Of(0) == c.Of(255) {
		t.Errorf("the classes of 0..255 split at 250 are %d, and 255 is in %d", c.Len(), c.Of(255))
	}
}

func TestAStateKnowsItsRules(t *testing.T) {
	one := func(x uint16) automaton.Expr[uint16] { return automaton.Lit(automaton.One(x)) }
	d, err := automaton.Build([]automaton.Rule[uint16]{
		{Name: "a", Expr: one(1)},
		{Name: "ab", Expr: automaton.Cat(one(1), one(2))},
		{Name: "a+", Expr: automaton.Plus(one(1))},
	})
	if err != nil {
		t.Fatal(err)
	}
	at := d.Step(d.Start(), 1)
	if got := d.Done(at); len(got) != 2 || got[0] != 1 || got[1] != 3 {
		t.Errorf("after 1, done is %v; want a and a+", got)
	}
	if got := d.Going(at); len(got) != 2 || got[0] != 2 || got[1] != 3 {
		t.Errorf("after 1, going is %v; want ab and a+", got)
	}
	if d.Accept(at) != 1 {
		t.Errorf("after 1, the winner is %d; want a", d.Accept(at))
	}
}
