package gen

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
		f, err := parse(strings.NewReader(preludeSrc), "prelude.lex", true)

		if err != nil {
			panic("gen: the prelude does not parse: " + err.Error())
		}

		defs = f.Defs
	})

	return defs
}
