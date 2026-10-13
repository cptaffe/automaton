package dot

import (
	"bytes"
	"embed"
	"fmt"
	"slices"
	"strings"
	"text/template"
)

//go:embed graph.dot.tmpl
var files embed.FS

var draw = template.Must(template.New("graph.dot.tmpl").Funcs(template.FuncMap{
	"q":     func(s string) string { return quote(s, "n") },
	"brk":   func(s string) string { return quote(s, "l") },
	"flush": func(s string) string { return quote(strings.TrimSuffix(s, "\n")+"\n", "l") },
	"html":  func(s string) string { return strings.ReplaceAll(template.HTMLEscapeString(s), "\n", "<br/>") },

	"fail": func(look string) (string, error) {
		return "", fmt.Errorf("no look named %q", look)
	},
}).ParseFS(files, "graph.dot.tmpl"))

type Graph struct {
	Name string

	Note string

	Look string

	Compound bool

	Ranks []Rank

	Groups []Group
	Nodes  []Node
	Edges  []Edge
}

type Rank struct {
	Kind string
	IDs  []string
}

type Node struct {
	ID      string
	Label   string
	Caption string
	Shape   string
	Style   string

	Class string

	Twice bool

	Size string

	Bold bool

	Flush bool
}

type Edge struct {
	From   string
	To     string
	Label  string
	Class  string
	Dashed bool
	Dotted bool

	Invis bool

	Bold  bool
	Width string
	Loose bool
}

type Group struct {
	ID    string
	Label string
	Nodes []Node
}

func (g Graph) Bytes() ([]byte, error) {
	var b bytes.Buffer

	if err := draw.Execute(&b, g); err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}

func Ident(s string) string {
	var b strings.Builder

	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}

	return b.String()
}

func (n Node) Paint() string {
	switch {
	case slices.Contains(strings.Fields(n.Class), "start"):
		return `fillcolor="#1971c2", color="#1971c2", fontcolor="white"`
	case n.Twice:
		return `fillcolor="#ebfbee", color="#2f9e44", fontcolor="#2b8a3e"`
	}

	return `fillcolor="#f1f3f5", color="#adb5bd", fontcolor="#495057"`
}

func (e Edge) Paint() string {
	color := "#495057"

	for _, c := range strings.Fields(e.Class) {
		switch c {
		case "goto", "hands":
			color = "#7048e8"
		case "event":
			color = "#e8590c"
		case "cycle":
			color = "#1c7ed6"
		case "nest":
			return `color="#e03131", fontcolor="#e03131"`
		}
	}

	return fmt.Sprintf("color=%q, fontcolor=%q", color, color)
}
