package lr

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
)

type Rule struct {
	Name string
	Alts []string
	At   int
}

type Grammar struct {
	Rules []Rule
	path  string
	lhs   map[int]string
}

var (
	item = regexp.MustCompile(`^\t(\S+):\s*(.*?)\s*\((\d+)\)\s*$`)
	word = regexp.MustCompile(`[A-Za-z_][A-Za-z_0-9]*`)
)

func Read(at string) (*Grammar, error) {
	f, err := os.Open(at)

	if err != nil {
		return nil, err
	}

	defer f.Close()

	seen := map[int]bool{}
	lhs := map[int]string{}
	alts := map[string][]string{}
	first := map[string]int{}

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<16), 1<<22)

	for sc.Scan() {
		m := item.FindStringSubmatch(sc.Text())

		if m == nil {
			continue
		}

		n := 0
		fmt.Sscanf(m[3], "%d", &n)

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

	return &Grammar{Rules: out, path: at, lhs: lhs}, nil
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
