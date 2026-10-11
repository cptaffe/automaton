package lexgen

import (
	_ "embed"
	"strings"
	"sync"
)

//go:embed prelude.lex
var preludeSrc string

var (
	once sync.Once
	defs []Def
)

func prelude() []Def {
	once.Do(func() {
		bare = true
		defer func() { bare = false }()

		f, err := Parse(strings.NewReader(preludeSrc), "prelude.lex")

		if err != nil {

			panic("lexgen: the prelude does not parse: " + err.Error())
		}

		defs = f.Defs
	})

	return defs
}

var bare bool
