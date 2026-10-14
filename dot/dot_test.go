package dot_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/cptaffe/automaton/dot"
	"github.com/cptaffe/automaton/dot/dottest"
)

type toy struct {
	name   string
	note   string
	starts []dot.Start
	moves  map[int][]dot.Move
	accept map[int]string
	leaves map[int]string
	states int
}

func (t toy) Name() string                { return t.name }
func (t toy) Note() string                { return t.note }
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
		note:   "toy -- 3 states",
		states: 3,
		starts: []dot.Start{{Name: "start", State: 1}},
		moves:  map[int][]dot.Move{1: {{To: 2, Label: "a-z"}, {To: 3, Label: "0-9", Dashed: true}}},
		accept: map[int]string{2: "word", 3: "number"},
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

type picture struct {
	note   string
	groups []string
	nodes  []string
	edges  []string
}

func look(g dot.Graph) picture {
	return picture{note: g.Note, groups: dottest.Groups(g), nodes: dottest.Nodes(g), edges: dottest.Edges(g)}
}

func TestADrawingHasANodePerStateAndAnEdgePerMove(t *testing.T) {
	draw := func(ms ...dot.Machine) dot.Graph { return dot.Draw(ms[0]) }

	whole := func(ms ...dot.Machine) dot.Graph { return dot.Whole(ms, "whole") }

	twoStarts := words()

	twoStarts.starts = []dot.Start{{Name: "b", State: 1}, {Name: "a", State: 1}, {Name: "c", State: 3}}

	unlabelled := words()

	unlabelled.accept = map[int]string{2: "", 3: "number"}

	noStart := words()

	noStart.starts = nil

	leaving := words()

	leaving.leaves = map[int]string{3: "chain"}

	lost := words()

	lost.leaves = map[int]string{3: "nowhere"}

	tail := toy{
		name:   "tail",
		states: 3,
		starts: []dot.Start{{Name: "s", State: 1}},
		moves:  map[int][]dot.Move{1: {{To: 3, Label: "x"}}, 3: {{To: 2, Label: "x"}}},
		accept: map[int]string{2: "end"},
	}

	back := toy{
		name:   "back",
		states: 3,
		starts: []dot.Start{{Name: "s", State: 1}},
		moves:  map[int][]dot.Move{1: {{To: 3, Label: "a"}}, 2: {{To: 1, Label: "c"}}, 3: {{To: 2, Label: "b"}}},
		accept: map[int]string{3: "end"},
	}

	for _, c := range []struct {
		name  string
		given []dot.Machine
		when  func(...dot.Machine) dot.Graph
		then  picture
	}{
		{
			name:  "Draw names starts and labels accepting states",
			given: []dot.Machine{words()},
			when:  draw,
			then: picture{
				note:   "toy -- 3 states",
				groups: []string{},
				nodes:  []string{"start (start)", "2 word (accept)", "3 number (accept)"},
				edges:  []string{"start -a-z-> 2 word", "start -0-9-> 3 number"},
			},
		},
		{
			name:  "Draw names a state with two starts by the first",
			given: []dot.Machine{twoStarts},
			when:  draw,
			then: picture{
				note:   "toy -- 3 states",
				groups: []string{},
				nodes:  []string{"b (start)", "2 word (accept)", "c number (start, accept)"},
				edges:  []string{"b -a-z-> 2 word", "b -0-9-> c number"},
			},
		},
		{
			name:  "Draw numbers an accepting state with no label",
			given: []dot.Machine{unlabelled},
			when:  draw,
			then: picture{
				note:   "toy -- 3 states",
				groups: []string{},
				nodes:  []string{"start (start)", "2 (accept)", "3 number (accept)"},
				edges:  []string{"start -a-z-> 2", "start -0-9-> 3 number"},
			},
		},
		{
			name:  "Whole folds a run of one label into one edge",
			given: []dot.Machine{chain()},
			when:  whole,
			then: picture{
				groups: []string{"chain: chain (start), 4, 5 done (accept)"},
				nodes:  []string{"chain (start)", "4", "5 done (accept)"},
				edges:  []string{"chain -x ×3-> 4", "4 -y-> 5 done"},
			},
		},
		{
			name:  "Whole folds a run through the last state",
			given: []dot.Machine{tail},
			when:  whole,
			then: picture{
				groups: []string{"tail: tail (start), 2 end (accept)"},
				nodes:  []string{"tail (start)", "2 end (accept)"},
				edges:  []string{"tail -x ×2-> 2 end"},
			},
		},
		{
			name:  "Whole keeps a state where a run from the last state stops",
			given: []dot.Machine{back},
			when:  whole,
			then: picture{
				groups: []string{"back: back (start), 2, 3 end (accept)"},
				nodes:  []string{"back (start)", "2", "3 end (accept)"},
				edges:  []string{"back -a-> 3 end", "2 -c-> back", "3 end -b-> 2"},
			},
		},
		{
			name:  "Whole names only the first start after its machine",
			given: []dot.Machine{twoStarts},
			when:  whole,
			then: picture{
				groups: []string{"toy: toy (start), 2 word (accept), 3 number (accept)"},
				nodes:  []string{"toy (start)", "2 word (accept)", "3 number (accept)"},
				edges:  []string{"toy -a-z-> 2 word", "toy -0-9-> 3 number"},
			},
		},
		{
			name:  "Whole draws a machine with no start without one",
			given: []dot.Machine{noStart},
			when:  whole,
			then: picture{
				groups: []string{"toy: 1, 2 word (accept), 3 number (accept)"},
				nodes:  []string{"1", "2 word (accept)", "3 number (accept)"},
				edges:  []string{"1 -a-z-> 2 word", "1 -0-9-> 3 number"},
			},
		},
		{
			name:  "Whole hands a state off to the start of the machine it leaves for",
			given: []dot.Machine{leaving, chain()},
			when:  whole,
			then: picture{
				groups: []string{
					"toy: toy (start), 2 word (accept), 3 number (accept)",
					"chain: chain (start), 4, 5 done (accept)",
				},
				nodes: []string{"toy (start)", "2 word (accept)", "3 number (accept)", "chain (start)", "4", "5 done (accept)"},
				edges: []string{"toy -a-z-> 2 word", "toy -0-9-> 3 number", "chain -x ×3-> 4", "4 -y-> 5 done", "3 number --> chain (hands)"},
			},
		},
		{
			name:  "Whole draws no hand-off to a machine it was not given",
			given: []dot.Machine{lost, chain()},
			when:  whole,
			then: picture{
				groups: []string{
					"toy: toy (start), 2 word (accept), 3 number (accept)",
					"chain: chain (start), 4, 5 done (accept)",
				},
				nodes: []string{"toy (start)", "2 word (accept)", "3 number (accept)", "chain (start)", "4", "5 done (accept)"},
				edges: []string{"toy -a-z-> 2 word", "toy -0-9-> 3 number", "chain -x ×3-> 4", "4 -y-> 5 done"},
			},
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			got := look(c.when(c.given...))

			if !reflect.DeepEqual(got, c.then) {
				t.Errorf("got %q; want %q", got, c.then)
			}
		})
	}
}

func TestDrawDashesTheMovesThatAskForIt(t *testing.T) {
	given := words()

	g := dot.Draw(given)

	dashed := make([]bool, 0, len(g.Edges))

	for _, e := range g.Edges {
		dashed = append(dashed, e.Dashed)
	}

	if want := []bool{false, true}; !reflect.DeepEqual(dashed, want) {
		t.Errorf("dashed %v; want %v", dashed, want)
	}
}

func TestAShapeDrawsRulesAndTheirLoops(t *testing.T) {
	given := []dot.Rule{
		{Name: "e", Alts: []string{"N", "L e R"}, Start: true},
		{Name: "l", Alts: []string{"l C I", "I"}},
		{Name: "p", Alts: []string{"e l"}},
		{Name: "a", Alts: []string{"b X"}},
		{Name: "b", Alts: []string{"a Y", "Z"}},
		{Name: "c", Alts: []string{"c P c"}},
	}

	for _, c := range []struct {
		name  string
		given []dot.Rule
		when  func([]dot.Rule) []string
		then  []string
	}{
		{
			name:  "a rule is a node with its alternatives",
			given: given,
			when:  func(rs []dot.Rule) []string { return dottest.Nodes(dot.Shape(rs, "g")) },
			then:  []string{"e N L e R (start)", "l l C I I", "p e l", "a b X", "b a Y Z", "c c P c"},
		},
		{
			name:  "a reference is an edge, and a reference within a loop is a cycle",
			given: given,
			when:  func(rs []dot.Rule) []string { return dottest.Edges(dot.Shape(rs, "g")) },
			then: []string{
				"e N L e R --> e N L e R (cycle nest)",
				"l l C I I --> l l C I I (cycle)",
				"p e l --> e N L e R",
				"p e l --> l l C I I",
				"a b X --> b a Y Z (cycle)",
				"b a Y Z --> a b X (cycle)",
				"c c P c --> c c P c (cycle nest)",
			},
		},
		{
			name:  "Nested names the rules of loops that embed themselves",
			given: given,
			when:  dot.Nested,
			then:  []string{"e", "c"},
		},
		{
			name:  "Nested names no rules of a grammar without loops",
			given: []dot.Rule{{Name: "p", Alts: []string{"q q"}}, {Name: "q", Alts: []string{"N"}}},
			when:  dot.Nested,
			then:  nil,
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			if got := c.when(c.given); !reflect.DeepEqual(got, c.then) {
				t.Errorf("got %q; want %q", got, c.then)
			}
		})
	}
}

func TestEachDrawingIsReadByGraphviz(t *testing.T) {
	quoted := toy{
		name:   `say "hi"`,
		note:   "C:\\dir\nnext",
		states: 2,
		starts: []dot.Start{{Name: `a "b"`, State: 1}},
		moves:  map[int][]dot.Move{1: {{To: 2, Label: `"`}}},
		accept: map[int]string{2: ""},
	}

	plain := chain()

	plain.accept = map[int]string{5: ""}

	for _, c := range []struct {
		name  string
		given dot.Graph
		then  dottest.Drawing
	}{
		{
			name:  "Draw",
			given: dot.Draw(quoted),
			then: dottest.Drawing{
				Name:  `say "hi"`,
				Note:  `C:\\dir\lnext\l`,
				Nodes: []string{`s1 a "b"`, "s2 2"},
				Edges: []string{`s1 - " -> s2`},
			},
		},
		{
			name:  "Whole",
			given: dot.Whole([]dot.Machine{plain}, "whole"),
			then: dottest.Drawing{
				Name:  "whole",
				Boxes: []string{`chain\l: m0_s1, m0_s4, m0_s5`},
				Nodes: []string{"m0_s1 chain", "m0_s4 4", "m0_s5 5"},
				Edges: []string{"m0_s1 - x ×3 -> m0_s4", "m0_s4 - y -> m0_s5"},
			},
		},
		{
			name:  "Shape",
			given: dot.Shape([]dot.Rule{{Name: "e", Alts: []string{"N", "L e R"}, Start: true}}, "g"),
			then: dottest.Drawing{
				Name:  "g",
				Nodes: []string{`e e\l  N\l  L e R\l`},
				Edges: []string{"e --> e"},
			},
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			b, err := c.given.Bytes()
			if err != nil {
				t.Fatal(err)
			}

			if got := dottest.Parse(t, b); !reflect.DeepEqual(got, c.then) {
				t.Errorf("got %q; want %q", got, c.then)
			}
		})
	}
}

func TestAnUnknownLookIsAnError(t *testing.T) {
	given := dot.Graph{Name: "g", Look: "nonsense"}

	_, err := given.Bytes()

	if err == nil || !strings.Contains(err.Error(), `no look named "nonsense"`) {
		t.Errorf("err %v; want no look named \"nonsense\"", err)
	}
}

func TestPaint(t *testing.T) {
	node := func(class string) string { return dot.Node{Class: class}.Paint() }

	accept := func(class string) string { return dot.Node{Class: class, Twice: true}.Paint() }

	edge := func(class string) string { return dot.Edge{Class: class}.Paint() }

	for _, c := range []struct {
		name  string
		given string
		when  func(string) string
		then  string
	}{
		{"a state is grey", "", node, `fillcolor="#f1f3f5", color="#adb5bd", fontcolor="#495057"`},
		{"a start is blue", "start", node, `fillcolor="#1971c2", color="#1971c2", fontcolor="white"`},
		{"a start among other classes is blue", "group start", node, `fillcolor="#1971c2", color="#1971c2", fontcolor="white"`},
		{"a class that holds the word start is not a start", "started", node, `fillcolor="#f1f3f5", color="#adb5bd", fontcolor="#495057"`},
		{"an accepting state is green", "", accept, `fillcolor="#ebfbee", color="#2f9e44", fontcolor="#2b8a3e"`},
		{"an accepting start is blue", "start", accept, `fillcolor="#1971c2", color="#1971c2", fontcolor="white"`},
		{"a move is grey", "", edge, `color="#495057", fontcolor="#495057"`},
		{"a goto is violet", "goto", edge, `color="#7048e8", fontcolor="#7048e8"`},
		{"a hand-off is violet", "hands", edge, `color="#7048e8", fontcolor="#7048e8"`},
		{"an event is orange", "event", edge, `color="#e8590c", fontcolor="#e8590c"`},
		{"a cycle is blue", "cycle", edge, `color="#1c7ed6", fontcolor="#1c7ed6"`},
		{"a nesting cycle is red", "cycle nest", edge, `color="#e03131", fontcolor="#e03131"`},
		{"a nest is red whatever follows", "nest cycle", edge, `color="#e03131", fontcolor="#e03131"`},
	} {
		t.Run(c.name, func(t *testing.T) {
			if got := c.when(c.given); got != c.then {
				t.Errorf("got %s; want %s", got, c.then)
			}
		})
	}
}

func TestPlural(t *testing.T) {
	for _, c := range []struct {
		name  string
		given string
		when  int
		then  string
	}{
		{"none are plural", "state", 0, "0 states"},
		{"one is singular", "state", 1, "1 state"},
		{"many are plural", "state", 2, "2 states"},
		{"a word ending in s takes es", "class", 2, "2 classes"},
		{"one of a word ending in s takes nothing", "class", 1, "1 class"},
	} {
		t.Run(c.name, func(t *testing.T) {
			if got := dot.Plural(c.when, c.given); got != c.then {
				t.Errorf("Plural(%d, %q) = %q; want %q", c.when, c.given, got, c.then)
			}
		})
	}
}
