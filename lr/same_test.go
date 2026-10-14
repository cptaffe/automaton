package lr

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/cptaffe/automaton/fa"
)

func same(a, b *fa.DFA[int], words []string) (bool, string) {
	type pair struct{ x, y fa.State }

	seen := map[pair]bool{}
	word := map[pair][]string{}
	start := pair{a.Start(), b.Start()}
	work := []pair{start}
	seen[start] = true
	word[start] = nil

	for len(work) > 0 {
		p := work[0]
		work = work[1:]

		if (a.Accept(p.x) != fa.NoMatch) != (b.Accept(p.y) != fa.NoMatch) {
			return false, fmt.Sprintf("only one accepts after %v", word[p])
		}

		for i, name := range words {
			r := i + 1
			x, y := a.Step(p.x, r), b.Step(p.y, r)

			if (x == fa.NoState) != (y == fa.NoState) {
				return false, fmt.Sprintf("only one reads %s after %v", name, word[p])
			}

			if x == fa.NoState {
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

func yacc(t *testing.T, gram string) []byte {
	t.Helper()

	dir := t.TempDir()

	if err := os.WriteFile(filepath.Join(dir, "gram.y"), []byte(gram), 0o600); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("go", "tool", "goyacc", "-l", "-p", "yy", "-o", os.DevNull, "-v", filepath.Join(dir, "y.output"), filepath.Join(dir, "gram.y"))

	if got, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("goyacc: %v: %s", err, got)
	}

	out, err := os.ReadFile(filepath.Join(dir, "y.output"))

	if err != nil {
		t.Fatal(err)
	}

	return out
}

func TestBothConstructionsGiveTheExpectedMachine(t *testing.T) {
	for _, c := range []struct {
		name  string
		given string
		then  []string
	}{
		{
			"a list by right recursion",
			"%token ITEM COMMA\n%%\nlist: ITEM | ITEM COMMA list ;\n",
			[]string{"start 1", "0", "1 ITEM->2", "2 accept COMMA->1"},
		},
		{
			"lines by left recursion from nothing",
			"%token NEWLINE\n%%\nfile: lines ;\nlines: | lines NEWLINE ;\n",
			[]string{"start 1", "0", "1 accept NEWLINE->1"},
		},
		{
			"brackets that self-embed",
			"%token NUM LP RP\n%%\nexpr: NUM | LP expr RP ;\n",
			[]string{"start 1", "0", "1 NUM->2 LP->1", "2 accept RP->2"},
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			text := yacc(t, c.given)
			g, err := Read(bytes.NewReader(text))

			if err != nil {
				t.Fatal(err)
			}

			ss, err := g.states(bytes.NewReader(text))

			if err != nil {
				t.Fatal(err)
			}

			mine := g.Tokens()
			theirs := stackless(ss, mine.at)

			if got := rows(mine.dfa, mine.words); !reflect.DeepEqual(got, c.then) {
				t.Errorf("over the grammar the machine is\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(c.then, "\n"))
			}

			if got := rows(theirs, mine.words); !reflect.DeepEqual(got, c.then) {
				t.Errorf("from the LR states the machine is\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(c.then, "\n"))
			}
		})
	}
}
