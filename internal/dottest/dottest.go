package dottest

import (
	"bytes"
	"encoding/json"
	"os/exec"
	"strings"
	"testing"

	"github.com/cptaffe/automaton/dot"
)

func Nodes(g dot.Graph) []string {
	out := make([]string, 0, len(g.Nodes))

	for _, n := range nodes(g) {
		out = append(out, name(n)+kind(n))
	}

	return out
}

func Edges(g dot.Graph) []string {
	names := map[string]string{}

	for _, n := range nodes(g) {
		names[n.ID] = name(n)
	}

	out := make([]string, 0, len(g.Edges))

	for _, e := range g.Edges {
		s := names[e.From] + " -" + strings.TrimSpace(e.Label) + "-> " + names[e.To]

		if e.Class != "" {
			s += " (" + e.Class + ")"
		}

		out = append(out, s)
	}

	return out
}

func Groups(g dot.Graph) []string {
	out := make([]string, 0, len(g.Groups))

	for _, gr := range g.Groups {
		members := make([]string, 0, len(gr.Nodes))

		for _, n := range gr.Nodes {
			members = append(members, name(n)+kind(n))
		}

		out = append(out, strings.TrimSpace(gr.Label)+": "+strings.Join(members, ", "))
	}

	return out
}

func nodes(g dot.Graph) []dot.Node {
	out := append([]dot.Node(nil), g.Nodes...)

	for _, gr := range g.Groups {
		out = append(out, gr.Nodes...)
	}

	return out
}

func name(n dot.Node) string {
	return strings.Join(strings.Fields(strings.TrimSpace(n.Caption+"\n"+n.Label)), " ")
}

func kind(n dot.Node) string {
	switch {
	case n.Class != "" && n.Twice:
		return " (" + n.Class + ", accept)"
	case n.Class != "":
		return " (" + n.Class + ")"
	case n.Twice:
		return " (accept)"
	}

	return ""
}

type Drawing struct {
	Name  string
	Note  string
	Boxes []string
	Nodes []string
	Edges []string
}

func Parse(t testing.TB, b []byte) Drawing {
	t.Helper()

	path, err := exec.LookPath("dot")

	if err != nil {
		t.Skip("Graphviz is not installed")
	}

	var stdout, stderr bytes.Buffer

	cmd := exec.Command(path, "-Tjson0")
	cmd.Stdin, cmd.Stdout, cmd.Stderr = bytes.NewReader(b), &stdout, &stderr

	if err := cmd.Run(); err != nil || stderr.Len() > 0 {
		t.Fatalf("Graphviz does not read the drawing: %v %s\n%s", err, stderr.Bytes(), b)
	}

	var got struct {
		Name    string
		Label   string
		Objects []struct {
			ID    int `json:"_gvid"`
			Name  string
			Label string
			Nodes []int
		}
		Edges []struct {
			Tail, Head int
			Label      string
		}
	}

	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatal(err)
	}

	names := map[int]string{}

	for _, o := range got.Objects {
		names[o.ID] = o.Name
	}

	out := Drawing{Name: got.Name, Note: got.Label}

	for _, o := range got.Objects {
		if !strings.HasPrefix(o.Name, "cluster_") {
			out.Nodes = append(out.Nodes, o.Name+" "+o.Label)

			continue
		}

		members := make([]string, 0, len(o.Nodes))

		for _, n := range o.Nodes {
			members = append(members, names[n])
		}

		out.Boxes = append(out.Boxes, o.Label+": "+strings.Join(members, ", "))
	}

	for _, e := range got.Edges {
		out.Edges = append(out.Edges, names[e.Tail]+" -"+e.Label+"-> "+names[e.Head])
	}

	return out
}
