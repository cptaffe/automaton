package fa

type edge[S Symbol] struct {
	set  Set[S]
	to   int
	eps  bool
	weak bool
}

type nfa[S Symbol] struct {
	out    [][]edge[S]
	accept []Match
	owner  []Match
	rule   Match
}

func (n *nfa[S]) state() int {
	n.out = append(n.out, nil)
	n.accept = append(n.accept, NoMatch)
	n.owner = append(n.owner, n.rule)

	return len(n.out) - 1
}

func (n *nfa[S]) on(from int, s Set[S], to int) {
	n.out[from] = append(n.out[from], edge[S]{set: s, to: to})
}

func (n *nfa[S]) eps(from, to int) {
	n.out[from] = append(n.out[from], edge[S]{to: to, eps: true})
}

func (n *nfa[S]) build(e Expr[S], from int) int {
	switch e.kind {
	case empty:
		return from
	case lit:
		to := n.state()
		n.on(from, e.set, to)

		return to
	case fallback:
		to := n.state()
		n.out[from] = append(n.out[from], edge[S]{set: e.set, to: to, weak: true})

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

func (n *nfa[S]) closure(of []int) []int {
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

type NFA[S Symbol] struct {
	n     nfa[S]
	start int
}

func (a *NFA[S]) State() int { return a.n.state() }

func (a *NFA[S]) On(from int, s Set[S], to int) { a.n.on(from, s, to) }

func (a *NFA[S]) Eps(from, to int) { a.n.eps(from, to) }

func (a *NFA[S]) Accepts(s int, m Match) { a.n.accept[s] = m }

func (a *NFA[S]) Start(s int) { a.start = s }

func (a *NFA[S]) Sets() []Set[S] {
	var out []Set[S]

	for _, es := range a.n.out {
		for _, e := range es {
			if !e.eps {
				out = append(out, e.set)
			}
		}
	}

	return out
}

func (a *NFA[S]) DFA() (*DFA[S], error) { return a.n.determinise(a.start, Split(a.Sets()), 0) }
