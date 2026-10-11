package dot_test

import (
	"strings"
	"testing"

	"github.com/cptaffe/automaton/dot"
)

type toy struct {
	name   string
	starts []dot.Start
	moves  map[int][]dot.Move
	accept map[int]string
	leaves map[int]string
	states int
}

func (t toy) Name() string                { return t.name }
func (t toy) Note() string                { return "" }
func (t toy) States() int                 { return t.states }
func (t toy) Starts() []dot.Start         { return t.starts }
func (t toy) Moves(s int) []dot.Move      { return t.moves[s] }
func (t toy) Leaves(s int) (string, bool) { to, ok := t.leaves[s]; return to, ok }

func (t toy) Accepts(s int) (string, bool) {
	label, ok := t.accept[s]
	return label, ok
}

func words() toy {
	return toy{
		name:   "toy",
		states: 3,
		starts: []dot.Start{{Name: "start", State: 1}},
		moves:  map[int][]dot.Move{1: {{To: 2, Label: "a-z"}, {To: 3, Label: "0-9", Dashed: true}}},
		accept: map[int]string{2: "word", 3: "number"},
	}
}

func drawn(t *testing.T) func([]byte, error) string {
	return func(b []byte, err error) string {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
}

func TestAMoveIsAnEdge(t *testing.T) {
	out := drawn(t)(dot.Draw(words()))

	if n := strings.Count(out, "->"); n != 2 {
		t.Errorf("the drawing has %d edges; want 2:\n%s", n, out)
	}
	for _, want := range []string{`" a-z "`, `" 0-9 "`, "style=dashed", "doublecircle", "word", "number", `label="start"`} {
		if !strings.Contains(out, want) {
			t.Errorf("the drawing does not hold %s:\n%s", want, out)
		}
	}
}

func TestTheFirstStartNamesAShared(t *testing.T) {
	m := words()
	m.starts = []dot.Start{{Name: "b", State: 1}, {Name: "a", State: 1}}
	for range 20 {
		if out := drawn(t)(dot.Draw(m)); !strings.Contains(out, `label="b"`) {
			t.Fatalf("two starts on one state are drawn as the second:\n%s", out)
		}
	}
}

func chain() toy {
	return toy{
		name:   "chain",
		states: 5,
		starts: []dot.Start{{Name: "chain", State: 1}},
		moves: map[int][]dot.Move{
			1: {{To: 2, Label: "x"}},
			2: {{To: 3, Label: "x"}},
			3: {{To: 4, Label: "x"}},
			4: {{To: 5, Label: "y"}},
		},
		accept: map[int]string{5: "done"},
	}
}

func TestARunOfOneLabelIsFolded(t *testing.T) {
	out := drawn(t)(dot.Whole([]dot.Machine{chain()}, "whole"))

	if !strings.Contains(out, "x ×3") {
		t.Errorf("three x in a row are not folded into one edge:\n%s", out)
	}
	if n := strings.Count(out, "->"); n != 2 {
		t.Errorf("the folded chain has %d edges; want 2:\n%s", n, out)
	}
}

func TestAHandOffCrossesMachines(t *testing.T) {
	a := words()
	a.leaves = map[int]string{2: "chain"}
	out := drawn(t)(dot.Whole([]dot.Machine{a, chain()}, "whole"))

	if !strings.Contains(out, `m0_s2 -> m1_s1 [label="", class="hands", style=dashed, constraint=false]`) {
		t.Errorf("the hand-off from toy to chain is not drawn:\n%s", out)
	}
	if n := strings.Count(out, "subgraph cluster_"); n != 2 {
		t.Errorf("%d boxes for two machines:\n%s", n, out)
	}
}

func TestAShapeMarksEmbeddingCycles(t *testing.T) {
	rs := []dot.Rule{
		{Name: "expr", Alts: []string{"NUM", "LP expr RP"}, Start: true},
		{Name: "list", Alts: []string{"ITEM", "ITEM COMMA list"}},
	}
	out := drawn(t)(dot.Shape(rs, "g"))

	if !strings.Contains(out, `class="start"`) {
		t.Errorf("the first rule is not marked as the start:\n%s", out)
	}
	if got := dot.Nested(rs); len(got) != 1 || got[0] != "expr" {
		t.Errorf("Nested is %v; want [expr]", got)
	}
}

func TestAnUnknownLookIsAnError(t *testing.T) {
	if _, err := (dot.Graph{Name: "g", Look: "nonsense"}).Bytes(); err == nil {
		t.Error("an unknown look drew without complaint")
	}
}

func TestPlural(t *testing.T) {
	for _, c := range []struct {
		n    int
		what string
		want string
	}{{0, "state", "0 states"}, {1, "state", "1 state"}, {2, "class", "2 classes"}} {
		if got := dot.Plural(c.n, c.what); got != c.want {
			t.Errorf("Plural(%d, %q) = %q; want %q", c.n, c.what, got, c.want)
		}
	}
}
