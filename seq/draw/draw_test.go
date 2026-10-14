package draw_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/cptaffe/automaton/fa"
	"github.com/cptaffe/automaton/internal/dottest"
	"github.com/cptaffe/automaton/seq"
	"github.com/cptaffe/automaton/seq/draw"
	"github.com/cptaffe/automaton/seq/hotkey"
	"github.com/cptaffe/automaton/seq/keypad"
)

func table(accept map[int][]*seq.Accept, moves map[int]map[string]int) *seq.Table {
	a := seq.NewAlphabet([]string{"a", "b", "c", "d"}, "tick", "tock", "tack")
	sets := make([]fa.Set[seq.Sym], 0, a.Size()+1)
	sets = append(sets, fa.Span(0, seq.Sym(a.Size()-1)))
	for s := range a.Size() {
		sets = append(sets, fa.One(seq.Sym(s)))
	}
	cs := fa.Partition(sets)
	states := make([]seq.State, 4)
	for s := range states {
		states[s].Next = make([]int, cs.Len()+1)
		for key, to := range moves[s] {
			sym, _ := a.Of(key)
			states[s].Next[cs.Of(sym)] = to
		}
		states[s].Accept = accept[s]
	}
	return &seq.Table{
		Name:    "t",
		Alpha:   a,
		Classes: cs,
		Scopes:  []seq.Scope{{Name: "main", Start: 1}, {Name: "other", Start: 3}},
		States:  states,
	}
}

func TestAMoveIsNamedByItsKeysAndEvents(t *testing.T) {
	for _, c := range []struct {
		name  string
		given map[int][]*seq.Accept
		when  map[int]map[string]int
		then  []string
	}{
		{
			name: "a run of three keys is a range",
			when: map[int]map[string]int{1: {"a": 2, "b": 2, "c": 2}},
			then: []string{"main -a-c-> 2"},
		},
		{
			name: "a run of two keys is named key by key",
			when: map[int]map[string]int{1: {"a": 2, "b": 2}},
			then: []string{"main -a b-> 2"},
		},
		{
			name: "keys apart are named apart",
			when: map[int]map[string]int{1: {"a": 2, "c": 2}},
			then: []string{"main -a c-> 2"},
		},
		{
			name: "every key to one target is any",
			when: map[int]map[string]int{1: {"a": 2, "b": 2, "c": 2, "d": 2}},
			then: []string{"main -any-> 2"},
		},
		{
			name: "the target of most keys is any when every key moves",
			when: map[int]map[string]int{1: {"a": 3, "b": 2, "c": 3, "d": 3}},
			then: []string{"main -b-> 2", "main -any-> other"},
		},
		{
			name: "the target of half the keys is not any",
			when: map[int]map[string]int{1: {"a": 2, "b": 2, "c": 3, "d": 3}},
			then: []string{"main -a b-> 2", "main -c d-> other"},
		},
		{
			name: "no target is any when a key does not move",
			when: map[int]map[string]int{1: {"a": 2, "b": 2, "c": 2}, 2: {"d": 3}},
			then: []string{"main -a-c-> 2", "2 -d-> other"},
		},
		{
			name: "an event is drawn in brackets as an event",
			when: map[int]map[string]int{1: {"tick": 2}},
			then: []string{"main -<tick>-> 2 (event)"},
		},
		{
			name: "a run of events is named event by event",
			when: map[int]map[string]int{1: {"tick": 2, "tock": 2, "tack": 2}},
			then: []string{"main -<tick> <tock> <tack>-> 2 (event)"},
		},
		{
			name: "a key with an event is not an event",
			when: map[int]map[string]int{1: {"d": 2, "tick": 2}},
			then: []string{"main -d <tick>-> 2"},
		},
		{
			name: "a goto is drawn once for each scope it goes to",
			given: map[int][]*seq.Accept{2: {
				{Action: "x", Goto: 2},
				{Action: "y", Goto: 2},
				{Action: "z", Goto: 1},
				{Action: "w"},
			}},
			when: map[int]map[string]int{1: {"a": 2}},
			then: []string{"main -a-> 2 x y z w", "2 x y z w --> other (goto)", "2 x y z w --> main (goto)"},
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			if got := dottest.Edges(draw.Draw(table(c.given, c.when))); !reflect.DeepEqual(got, c.then) {
				t.Errorf("got %q; want %q", got, c.then)
			}
		})
	}
}

func TestAStateIsNamedByItsScopeOrWhatItAccepts(t *testing.T) {
	given := table(map[int][]*seq.Accept{2: {{Action: "x", When: seq.Name("on")}, {Action: "y"}}}, nil)

	g := draw.Draw(given)
	label, ok := draw.Machine(given).Accepts(2)
	_, none := draw.Machine(given).Accepts(1)

	if want := "t -- 3 states, 7 classes"; g.Note != want {
		t.Errorf("note %q; want %q", g.Note, want)
	}
	if want := []string{"main (start)", "2 x when on y (accept)", "other (start)"}; !reflect.DeepEqual(dottest.Nodes(g), want) {
		t.Errorf("nodes %q; want %q", dottest.Nodes(g), want)
	}
	if want := "x\nwhen on\ny"; label != want || !ok || none {
		t.Errorf("Accepts gives %q, %v and %v; want %q, true and false", label, ok, none, want)
	}
}

func TestADialPlanIsDrawnWhole(t *testing.T) {
	tree, err := keypad.Parse(strings.NewReader("%delay T 5s\n\nstart:\n  0           do operator -> start\n  NXX T       do short when on\n  960-X       do long\n"), "plan.tree")
	if err != nil {
		t.Fatal(err)
	}
	given, err := keypad.Compile(tree)
	if err != nil {
		t.Fatal(err)
	}

	g := draw.Draw(given)

	if want := "plan -- 10 states, 7 classes"; g.Note != want {
		t.Errorf("note %q; want %q", g.Note, want)
	}
	if want := []string{"start (start)", "2 operator (accept)", "3", "4", "5", "6", "7", "8", "9 short when on (accept)", "10 long (accept)"}; !reflect.DeepEqual(dottest.Nodes(g), want) {
		t.Errorf("nodes %q; want %q", dottest.Nodes(g), want)
	}
	want := []string{
		"start -0-> 2 operator",
		"start -2-8-> 3",
		"start -9-> 4",
		"2 operator --> start (goto)",
		"3 -0-9-> 5",
		"4 -0-5 7-9-> 5",
		"4 -6-> 6",
		"5 -0-9-> 7",
		"6 -1-9-> 7",
		"6 -0-> 8",
		"7 -<T>-> 9 short when on (event)",
		"8 -<T>-> 9 short when on (event)",
		"8 -0-9-> 10 long",
	}
	if !reflect.DeepEqual(dottest.Edges(g), want) {
		t.Errorf("edges %q; want %q", dottest.Edges(g), want)
	}
}

func TestAHotkeyFallbackIsDrawnAsAny(t *testing.T) {
	r, err := hotkey.Parse(strings.NewReader("main:\n\tcontrol-x control-s { put }\n\tcontrol-x any { }\n"), "h", nil)
	if err != nil {
		t.Fatal(err)
	}

	g := draw.Draw(r.Table)

	if want := []string{"main -control-x-> 2", "2 -any-> 3", "2 -control-s-> 4 put"}; !reflect.DeepEqual(dottest.Edges(g), want) {
		t.Errorf("edges %q; want %q", dottest.Edges(g), want)
	}
}

func TestADrawingIsReadByGraphviz(t *testing.T) {
	given := table(map[int][]*seq.Accept{2: {{Action: `say "hi" & <go>`, Goto: 2}}}, map[int]map[string]int{1: {"a": 2, "tick": 3}})

	b, err := draw.Draw(given).Bytes()
	if err != nil {
		t.Fatal(err)
	}
	got := dottest.Parse(t, b)

	want := dottest.Drawing{
		Name:  "t",
		Note:  `t -- 3 states, 7 classes\l`,
		Nodes: []string{"s1 main", `s2 <font point-size="7" color="#868e96">2</font><br/>say &#34;hi&#34; &amp; &lt;go&gt;`, "s3 other"},
		Edges: []string{"s1 - a -> s2", "s1 - <tick> -> s3", "s2 -  -> s3"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %q; want %q", got, want)
	}
}

func TestLabel(t *testing.T) {
	for _, c := range []struct {
		name  string
		given string
		when  seq.Cond
		then  string
	}{
		{"a short action is itself", "put", nil, "put"},
		{"an action is trimmed", "  put\n", nil, "put"},
		{"an action of 24 bytes is whole", "abcdefghijklmnopqrstuvwx", nil, "abcdefghijklmnopqrstuvwx"},
		{"a longer action is cut to 21 bytes", "abcdefghijklmnopqrstuvwxy", nil, "abcdefghijklmnopqrstu..."},
		{"a cut drops the space it ends on", "abcdefghijklmnopqrst uvwxyz", nil, "abcdefghijklmnopqrst..."},
		{"only the first line of an action is shown", "a\nb", nil, "a..."},
		{"a condition follows the action", "put", seq.Name("on"), "put\nwhen on"},
		{"a condition stands alone without an action", "", seq.Name("on"), "when on"},
		{"a condition of 14 bytes is one line", "", seq.And{seq.Name("on"), seq.Name("up")}, "when on and up"},
		{"a condition of 15 bytes wraps", "", seq.And{seq.Name("on"), seq.Name("upx")}, "when on and\nupx"},
		{"a longer condition wraps at 14 bytes", "", seq.And{seq.Name("on"), seq.Name("up"), seq.Name("in")}, "when on and up\nand in"},
		{"a long word has a line of its own", "", seq.Name("averyveryverylongname"), "when\naveryveryverylongname"},
	} {
		t.Run(c.name, func(t *testing.T) {
			if got := draw.Label(&seq.Accept{Action: c.given, When: c.when}); got != c.then {
				t.Errorf("got %q; want %q", got, c.then)
			}
		})
	}
}
