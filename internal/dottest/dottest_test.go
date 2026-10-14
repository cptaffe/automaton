package dottest_test

import (
	"reflect"
	"testing"

	"github.com/cptaffe/automaton/dot"
	"github.com/cptaffe/automaton/internal/dottest"
)

func TestAGraphIsWrittenAsLines(t *testing.T) {
	given := dot.Graph{
		Nodes: []dot.Node{
			{ID: "a", Label: "start", Class: "start"},
			{ID: "b", Caption: "2", Label: "word\nwhen on", Twice: true},
		},
		Groups: []dot.Group{{Label: "box\n", Nodes: []dot.Node{{ID: "c", Label: "3"}, {ID: "d", Label: "end", Class: "start", Twice: true}}}},
		Edges: []dot.Edge{
			{From: "a", To: "b", Label: " a-z "},
			{From: "b", To: "c", Class: "goto"},
		},
	}

	nodes, edges, groups := dottest.Nodes(given), dottest.Edges(given), dottest.Groups(given)

	if want := []string{"start (start)", "2 word when on (accept)", "3", "end (start, accept)"}; !reflect.DeepEqual(nodes, want) {
		t.Errorf("nodes %q; want %q", nodes, want)
	}
	if want := []string{"start -a-z-> 2 word when on", "2 word when on --> 3 (goto)"}; !reflect.DeepEqual(edges, want) {
		t.Errorf("edges %q; want %q", edges, want)
	}
	if want := []string{"box: 3, end (start, accept)"}; !reflect.DeepEqual(groups, want) {
		t.Errorf("groups %q; want %q", groups, want)
	}
}

func TestParseReadsADrawingAsGraphvizDoes(t *testing.T) {
	given := []byte(`digraph "g" {
  label="note \"q\"\l"
  subgraph cluster_0 { label="box"; c [label="c"] }
  a [label="say \"hi\"\nC:\\d"]
  a -> c [label=" a-z "]
}
`)

	got := dottest.Parse(t, given)

	want := dottest.Drawing{
		Name:  "g",
		Note:  `note "q"\l`,
		Boxes: []string{"box: c"},
		Nodes: []string{"c c", `a say "hi"\nC:\\d`},
		Edges: []string{"a - a-z -> c"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %q; want %q", got, want)
	}
}
