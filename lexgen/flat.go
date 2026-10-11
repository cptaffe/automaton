package lexgen

type FlatActs struct {
	Emit string
	Do   string
	Goto string
	Push string
	Pop  bool
	Drop bool
}

type FlatRule struct {
	Expr  Expr
	Acts  FlatActs
	Scope string
	Line  int
	Text  string
}

type FlatScope struct {
	Name     string
	Includes string
	Line     int
}

type Flat struct {
	Defs     []Def
	Name     string
	Scopes   []FlatScope
	Rules    []FlatRule
	Shadowed []string
}

func (f *Flat) Scope(name string) *FlatScope {
	for i := range f.Scopes {
		if f.Scopes[i].Name == name {
			return &f.Scopes[i]
		}
	}

	return nil
}

func (f *Flat) Of(name string) []FlatRule {
	var out []FlatRule
	seen := map[string]bool{}

	for at := name; at != "" && !seen[at]; {
		seen[at] = true

		for _, r := range f.Rules {
			if r.Scope == at {
				out = append(out, r)
			}
		}

		s := f.Scope(at)

		if s == nil {
			break
		}

		at = s.Includes
	}

	return out
}
