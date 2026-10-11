package dot

import (
	"bytes"
	"embed"
	"fmt"
	"strings"
	"text/template"
)

//go:embed graph.dot.tmpl
var files embed.FS

var draw = template.Must(template.New("graph.dot.tmpl").Funcs(template.FuncMap{
	"q":     func(s string) string { return quote(s, "n") },
	"brk":   func(s string) string { return quote(s, "l") },
	"flush": func(s string) string { return quote(strings.TrimSuffix(s, "\n")+"\n", "l") },

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
	ID    string
	Label string
	Shape string
	Style string

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

func (g Graph) Bytes() []byte {
	var b bytes.Buffer

	if err := draw.Execute(&b, g); err != nil {
		panic(err)
	}

	return b.Bytes()
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
