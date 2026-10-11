package lr

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/cptaffe/automaton"
)

func same(a, b *automaton.DFA, words []string) (bool, string) {
	type pair struct{ x, y automaton.State }

	seen := map[pair]bool{}
	word := map[pair][]string{}
	start := pair{a.Start(), b.Start()}
	work := []pair{start}
	seen[start] = true
	word[start] = nil

	for len(work) > 0 {
		p := work[0]
		work = work[1:]

		if (a.Accept(p.x) != automaton.NoMatch) != (b.Accept(p.y) != automaton.NoMatch) {
			return false, fmt.Sprintf("only one accepts after %v", word[p])
		}

		for i, name := range words {
			r := rune(i + 1)
			x, y := a.Step(p.x, r), b.Step(p.y, r)

			if (x == automaton.NoState) != (y == automaton.NoState) {
				return false, fmt.Sprintf("only one reads %s after %v", name, word[p])
			}

			if x == automaton.NoState {
				continue
			}

			q := pair{x, y}

			if seen[q] {
				continue
			}

			seen[q] = true
			word[q] = append(append([]string(nil), word[p]...), name)
			work = append(work, q)
		}
	}

	return true, ""
}

func TestTheDrawingIsTheLRAutomatonWithoutItsStack(t *testing.T) {
	for _, dir := range []string{
		"../lex/gen",
		"../seq/keypad",
		"../seq/hotkey",
	} {
		name := filepath.Base(dir)

		t.Run(name, func(t *testing.T) {
			out := filepath.Join(t.TempDir(), "y.output")
			cmd := exec.Command("go", "tool", "goyacc", "-l", "-p", "yy", "-o", os.DevNull, "-v", out, "gram.y")
			cmd.Dir = dir

			if got, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("goyacc: %v: %s", err, got)
			}

			text, err := os.ReadFile(out)

			if err != nil {
				t.Fatal(err)
			}

			g, err := Read(bytes.NewReader(text))

			if err != nil {
				t.Fatal(err)
			}

			if len(g.Rules) == 0 {
				t.Fatal("no rules in y.output: this reads the one golang.org/x/tools/cmd/goyacc writes")
			}

			ss, err := g.states(bytes.NewReader(text))

			if err != nil {
				t.Fatal(err)
			}

			mine := g.Tokens()

			if mine == nil {
				t.Fatal("no machine over tokens")
			}

			theirs := stackless(ss, mine.at)

			if ok, why := same(mine.dfa, theirs, mine.words); !ok {
				t.Errorf("the two constructions differ: %s", why)
			}
		})
	}
}
