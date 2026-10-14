package lr

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type Rule struct {
	Name string
	Alts []string
	At   int
}

type Grammar struct {
	Rules []Rule
	lhs   map[int]string
}

var item = regexp.MustCompile(`^\t(\S+):\s*(.*?)\s*\((\d+)\)\s*$`)

func Read(r io.Reader) (*Grammar, error) {
	seen := map[int]bool{}
	lhs := map[int]string{}
	alts := map[string][]string{}
	first := map[string]int{}

	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 1<<16), 1<<22)

	for sc.Scan() {
		m := item.FindStringSubmatch(sc.Text())

		if m == nil {
			continue
		}

		n, err := strconv.Atoi(m[3])

		if err != nil {
			return nil, fmt.Errorf("rule %s: %w", m[3], err)
		}

		if seen[n] || m[1] == "$accept" {
			continue
		}

		seen[n] = true
		name, body := m[1], clean(m[2])

		if at, ok := first[name]; !ok || n < at {
			first[name] = n
		}

		alts[name] = append(alts[name], body)
		lhs[n] = name
	}

	var out []Rule

	for name := range alts {
		out = append(out, Rule{Name: name, Alts: alts[name], At: first[name]})
	}

	sort.Slice(out, func(i, j int) bool { return out[i].At < out[j].At })

	if err := sc.Err(); err != nil {
		return nil, err
	}

	if len(out) == 0 {
		return nil, errors.New("no grammar rules: this is not a report goyacc wrote with -v")
	}

	return &Grammar{Rules: out, lhs: lhs}, nil
}

func clean(s string) string {
	out := []string{}

	for _, w := range strings.Fields(strings.TrimSuffix(s, ".")) {
		if w == "." {
			continue
		}

		out = append(out, w)
	}

	if len(out) == 0 {
		return "nothing"
	}

	return strings.Join(out, " ")
}
