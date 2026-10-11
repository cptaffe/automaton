package automaton

type edge struct {
	set Set
	to  int
	eps bool
}

type nfa struct {
	out    [][]edge
	accept []Match
}

func (n *nfa) state() int {
	n.out = append(n.out, nil)
	n.accept = append(n.accept, NoMatch)

	return len(n.out) - 1
}

func (n *nfa) on(from int, s Set, to int) {
	n.out[from] = append(n.out[from], edge{set: s, to: to})
}

func (n *nfa) eps(from, to int) {
	n.out[from] = append(n.out[from], edge{to: to, eps: true})
}

func (n *nfa) build(e Expr, from int) int {
	switch e.kind {
	case empty:
		return from
	case lit:
		to := n.state()
		n.on(from, e.set, to)

		return to
	case cat:
		at := from

		for _, s := range e.sub {
			at = n.build(s, at)
		}

		return at
	case alt:
		to := n.state()

		for _, s := range e.sub {
			in := n.state()
			n.eps(from, in)
			n.eps(n.build(s, in), to)
		}

		return to
	case plus:
		in := n.state()
		n.eps(from, in)
		end := n.build(e.sub[0], in)
		n.eps(end, in)

		return end
	case star:
		in := n.state()
		to := n.state()
		n.eps(from, in)
		n.eps(from, to)
		end := n.build(e.sub[0], in)
		n.eps(end, in)
		n.eps(end, to)

		return to
	}

	return from
}

func (n *nfa) closure(of []int) []int {
	seen := make([]bool, len(n.out))
	todo := append([]int(nil), of...)
	var out []int

	for len(todo) > 0 {
		s := todo[len(todo)-1]
		todo = todo[:len(todo)-1]

		if seen[s] {
			continue
		}

		seen[s] = true
		out = append(out, s)

		for _, e := range n.out[s] {
			if e.eps {
				todo = append(todo, e.to)
			}
		}
	}

	return out
}

type NFA struct {
	n     nfa
	start int
	begun bool
}

func (a *NFA) State() int {
	s := a.n.state()

	if !a.begun {
		a.start, a.begun = s, true
	}

	return s
}

func (a *NFA) On(from int, s Set, to int) { a.n.on(from, s, to) }

func (a *NFA) Eps(from, to int) { a.n.eps(from, to) }

func (a *NFA) Accepts(s int, m Match) { a.n.accept[s] = m }

func (a *NFA) Start(s int) { a.start, a.begun = s, true }

func (a *NFA) Sets() []Set {
	var out []Set

	for _, es := range a.n.out {
		for _, e := range es {
			if !e.eps {
				out = append(out, e.set)
			}
		}
	}

	return out
}

func (a *NFA) Determinise(classes *Classes) (*DFA, error) {
	if err := classes.refine(a.Sets()); err != nil {
		return nil, err
	}

	return a.n.determinise(a.start, classes), nil
}

func (a *NFA) DFA() *DFA { return a.n.determinise(a.start, Partition(a.Sets())) }
