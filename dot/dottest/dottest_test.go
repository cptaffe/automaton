package dottest_test

import (
	"reflect"
	"testing"

	"github.com/cptaffe/automaton/dot"
	"github.com/cptaffe/automaton/dot/dottest"
)

func TestAGraphIsWrittenAsLines(t *testing.T) {
	given := dot.Graph{
		Nodes: []dot.Node{
			{ID: "a", Label: "start", Class: "start"},
			{ID: "b", Caption: "2", Label: "word\nwhen on", Twice: true},
		},
		Groups: []dot.Group{{Nodes: []dot.Node{{ID: "c", Label: "3"}}}},
		Edges: []dot.Edge{
			{From: "a", To: "b", Label: " a-z "},
			{From: "b", To: "c", Class: "goto"},
		},
	}

	nodes, edges := dottest.Nodes(given), dottest.Edges(given)

	if want := []string{"start (start)", "2 word when on (accept)", "3"}; !reflect.DeepEqual(nodes, want) {
		t.Errorf("nodes %q; want %q", nodes, want)
	}
	if want := []string{"start -a-z-> 2 word when on", "2 word when on --> 3 (goto)"}; !reflect.DeepEqual(edges, want) {
		t.Errorf("edges %q; want %q", edges, want)
	}
}
