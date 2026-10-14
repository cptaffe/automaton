package fa_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/cptaffe/automaton/fa"
)

func TestAnNFAStartsAtItsFirstStateUnlessTold(t *testing.T) {
	var n fa.NFA[rune]
	x, y, z := n.State(), n.State(), n.State()
	n.On(x, fa.One('a'), y)
	n.Eps(y, z)
	n.On(z, fa.One('b'), y)
	n.Accepts(y, 1)

	first := table(n.DFA())
	n.Start(z)
	moved := table(n.DFA())

	if then := []string{
		"start 1",
		"0 done [] going []",
		"1 done [] going [] a->2",
		"2 done [1] going [] b->2",
	}; !reflect.DeepEqual(first, then) {
		t.Errorf("from its first state the machine is\n%s\nwant\n%s", strings.Join(first, "\n"), strings.Join(then, "\n"))
	}

	if then := []string{
		"start 1",
		"0 done [] going []",
		"1 done [] going [] b->2",
		"2 done [1] going [] b->2",
	}; !reflect.DeepEqual(moved, then) {
		t.Errorf("from its third state the machine is\n%s\nwant\n%s", strings.Join(moved, "\n"), strings.Join(then, "\n"))
	}
}

func TestAnNFAListsTheSetsItsEdgesRead(t *testing.T) {
	var n fa.NFA[int]
	x, y := n.State(), n.State()
	n.On(x, fa.Span(1, 3), y)
	n.Eps(y, x)
	n.On(y, fa.One(7), x)

	got := n.Sets()

	if then := []fa.Set[int]{fa.Span(1, 3), fa.One(7)}; !reflect.DeepEqual(got, then) {
		t.Errorf("the sets are %v; want %v", got, then)
	}
}

func TestMinimisingDropsAStateThatNeverAccepts(t *testing.T) {
	var n fa.NFA[rune]
	x, sink, end := n.State(), n.State(), n.State()
	n.On(x, fa.One('a'), sink)
	n.On(x, fa.One('b'), end)
	n.Accepts(end, 1)
	d := n.DFA()

	got := table(d.Minimise())

	then := []string{
		"start 1",
		"0 done [] going []",
		"1 done [] going [] b->2",
		"2 done [1] going []",
	}

	if !reflect.DeepEqual(got, then) {
		t.Errorf("minimised, the machine is\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(then, "\n"))
	}
}
