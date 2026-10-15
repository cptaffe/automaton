package hotkey

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/cptaffe/automaton/internal/regextest"
	"github.com/cptaffe/automaton/internal/seqtest"
)

type refRule struct {
	lead   string
	middle *regexp.Regexp
	action string
}

type reference struct {
	rules   []refRule
	pending []string
}

func (r *reference) completes(w []string) string {
	for _, x := range r.rules {
		if len(w) >= 2 && w[0] == x.lead && w[len(w)-1] == "return" && x.middle.MatchString(strings.Join(w[1:len(w)-1], "")) {
			return x.action
		}
	}
	return ""
}

func (r *reference) continues(w []string) bool {
	for _, x := range r.rules {
		if w[0] != x.lead {
			continue
		}
		middle := strings.Join(w[1:], "")
		if len(middle) != len(w)-1 || strings.ContainsAny(middle, "xyz") {
			continue
		}
		for _, more := range regextest.Words(6) {
			if x.middle.MatchString(middle + more) {
				return true
			}
		}
	}
	return false
}

func (r *reference) read(w []string) (bool, string) {
	if a := r.completes(w); a != "" {
		r.pending = nil
		return true, a
	}
	if r.continues(w) {
		r.pending = w
		return true, ""
	}
	return false, ""
}

func (r *reference) press(key string) seqtest.Step {
	taken, fired := r.read(append(append([]string(nil), r.pending...), key))
	if !taken && len(r.pending) > 0 {
		r.pending = nil
		taken, fired = r.read([]string{key})
	}
	return seqtest.Step{Key: key, Taken: taken, Fired: fired, Pending: len(r.pending) > 0}
}

var keys = map[rune]string{'a': "a", 'b': "b", 'c': "c", 'x': "x", 'y': "y", 'z': "z", 'r': "return"}

func FuzzMachineAgreesWithAReference(f *testing.F) {
	f.Add([]byte{1, 0, 1, 0, 2}, []byte{2, 0, 4}, []byte{4, 0, 2}, "xabrybyxar")
	f.Add([]byte{0, 1}, []byte{1, 0, 2, 0, 4}, []byte{5, 0, 1, 0, 3}, "zaccrxazbr")
	f.Fuzz(func(t *testing.T, one, two, three []byte, typed string) {
		var src strings.Builder
		ref := &reference{}
		src.WriteString("main:\n")
		for i, data := range [][]byte{one, two, three} {
			p := regextest.Decode(data)
			x := refRule{lead: string("xyz"[i]), middle: regexp.MustCompile("^(?:" + p.Regexp + ")$"), action: fmt.Sprintf("r%d", i+1)}
			fmt.Fprintf(&src, "\t%s %s return { %s }\n", x.lead, p.Hotkey, x.action)
			ref.rules = append(ref.rules, x)
		}
		r, err := Parse(strings.NewReader(src.String()), "hotkeys", nil)
		if err != nil {
			t.Fatalf("%v\n%s", err, src.String())
		}
		m := seqtest.Machine(t, r.Table, "")
		var steps []seqtest.Step
		for _, k := range typed {
			if key, ok := keys[k]; ok && len(steps) < 10 {
				steps = append(steps, ref.press(key))
			}
		}

		seqtest.Run(t, m, r.Table.Alpha, nil, steps)

		if t.Failed() {
			t.Logf("rules:\n%s", src.String())
		}
	})
}
