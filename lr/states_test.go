package lr

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
)

type move struct {
	on   string
	to   int
	jump bool
}

type state struct {
	at      int
	items   []string
	moves   []move
	reduces []string
	accepts bool
}

var (
	head   = regexp.MustCompile(`^state (\d+)`)
	shift  = regexp.MustCompile(`^\t(\S+)\s+shift (\d+)$`)
	reduce = regexp.MustCompile(`^\t(\S+)\s+reduce (\d+)`)
	jump   = regexp.MustCompile(`^\t(\S+)\s+goto (\d+)$`)
	accept = regexp.MustCompile(`^\t\S+\s+accept$`)
	kernel = regexp.MustCompile(`^\t(\S+:\s.*?)\s*(?:\(\d+\))?\s*$`)
)

func (g *Grammar) states(r io.Reader) ([]state, error) {
	var out []state
	var now *state
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 1<<16), 1<<22)

	for sc.Scan() {
		line := sc.Text()

		if m := head.FindStringSubmatch(strings.TrimSpace(line)); m != nil {
			n, err := number(m[1])

			if err != nil {
				return nil, err
			}

			out = append(out, state{at: n})
			now = &out[len(out)-1]

			continue
		}

		if now == nil {
			continue
		}

		switch {
		case shift.MatchString(line):
			m := shift.FindStringSubmatch(line)
			to, err := number(m[2])

			if err != nil {
				return nil, err
			}

			now.moves = append(now.moves, move{on: m[1], to: to})
		case jump.MatchString(line):
			m := jump.FindStringSubmatch(line)
			to, err := number(m[2])

			if err != nil {
				return nil, err
			}

			now.moves = append(now.moves, move{on: m[1], to: to, jump: true})
		case reduce.MatchString(line):
			n, err := number(reduce.FindStringSubmatch(line)[2])

			if err != nil {
				return nil, err
			}

			if name, ok := g.lhs[n]; ok {
				now.reduces = append(now.reduces, name)
			}
		case accept.MatchString(line):
			now.accepts = true
		case kernel.MatchString(line):
			if m := kernel.FindStringSubmatch(line); !strings.Contains(m[1], ": .") {
				now.items = append(now.items, m[1])
			}
		}
	}

	return out, sc.Err()
}

func number(s string) (int, error) {
	n, err := strconv.Atoi(s)

	if err != nil {
		return 0, fmt.Errorf("y.output: %w", err)
	}

	return n, nil
}
