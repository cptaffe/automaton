package regextest

import "github.com/cptaffe/automaton/fa"

const Letters = "abc"

type Pattern struct {
	FA     fa.Expr[rune]
	Regexp string
	Lex    string
}

type decoder struct{ data []byte }

func Decode(data []byte) Pattern {
	return (&decoder{data}).pattern(0)
}

func (d *decoder) next() byte {
	if len(d.data) == 0 {
		return 0
	}
	b := d.data[0]
	d.data = d.data[1:]
	return b
}

func (d *decoder) pattern(depth int) Pattern {
	op := d.next() % 6
	if depth >= 3 {
		op = 0
	}
	switch op {
	case 1:
		a, b := d.pattern(depth+1), d.pattern(depth+1)
		return Pattern{fa.Cat(a.FA, b.FA), "(?:" + a.Regexp + ")(?:" + b.Regexp + ")", "(" + a.Lex + " " + b.Lex + ")"}
	case 2:
		a, b := d.pattern(depth+1), d.pattern(depth+1)
		return Pattern{fa.Alt(a.FA, b.FA), "(?:" + a.Regexp + "|" + b.Regexp + ")", "(" + a.Lex + " | " + b.Lex + ")"}
	case 3:
		a := d.pattern(depth + 1)
		return Pattern{fa.Star(a.FA), "(?:" + a.Regexp + ")*", "(" + a.Lex + ")*"}
	case 4:
		a := d.pattern(depth + 1)
		return Pattern{fa.Plus(a.FA), "(?:" + a.Regexp + ")+", "(" + a.Lex + ")+"}
	case 5:
		a := d.pattern(depth + 1)
		return Pattern{fa.Opt(a.FA), "(?:" + a.Regexp + ")?", "(" + a.Lex + ")?"}
	}
	mask := d.next()%7 + 1
	var runes []rune
	for i, r := range Letters {
		if mask&(1<<i) != 0 {
			runes = append(runes, r)
		}
	}
	set := "[" + string(runes) + "]"
	return Pattern{fa.Lit(fa.Text(string(runes))), set, set}
}

func Words(n int) []string {
	out := []string{""}
	for at := 0; at < len(out); at++ {
		if len(out[at]) < n {
			for _, r := range Letters {
				out = append(out, out[at]+string(r))
			}
		}
	}
	return out
}
