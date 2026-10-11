package lexgen

//go:generate go run ../cmd/genlex -go ../internal/lexlex/lex.gen.go -package lexlex lex.lex

import (
	"io"
	"strings"

	"github.com/cptaffe/automaton/internal/lexlex"
	"github.com/cptaffe/automaton/lexrun"
)

type parser struct {
	*lexrun.Reader

	named map[string]Expr

	given map[string]string
	order []string

	file *File
	ctx  string

	declLine int

	tok     int
	tokText string
}

func newParser(r io.Reader, from string) (*parser, error) {
	in, err := lexrun.Open(lexlex.Spec, r, "start", from, lexlex.TokNewline)

	if err != nil {
		return nil, err
	}

	p := &parser{Reader: in, named: predefined(), given: given(),
		file: &File{Name: in.Name()}}

	if !bare {
		for _, name := range builtinNames() {
			p.file.Defs = append(p.file.Defs, Def{Name: name, Expr: p.named[name]})
		}
	}

	return p, nil
}

func (p *parser) Error(string) {}

func (p *parser) Lex(v *yySymType) int {
	t, text := p.lex(v)
	p.tok, p.tokText = t, text

	return t
}

var kinds = map[int]int{
	lexlex.TokNewline:  NEWLINE,
	lexlex.TokArrow:    ARROW,
	lexlex.TokDefine:   DEFINE,
	lexlex.TokLbrace:   LBRACE,
	lexlex.TokRbrace:   RBRACE,
	lexlex.TokOr:       OR,
	lexlex.TokStar:     STAR,
	lexlex.TokPlus:     PLUS,
	lexlex.TokOpt:      OPT,
	lexlex.TokLparen:   LPAREN,
	lexlex.TokRparen:   RPAREN,
	lexlex.TokLbrack:   LBRACK,
	lexlex.TokRbrack:   RBRACK,
	lexlex.TokCaret:    CARET,
	lexlex.TokDash:     DASH,
	lexlex.TokMinus:    MINUS,
	lexlex.TokClass:    CLASS,
	lexlex.TokText:     TEXT,
	lexlex.TokAnycase:  ANYCASE,
	lexlex.TokIncludes: INCLUDES,
	lexlex.TokEmit:     EMIT,
	lexlex.TokIgnore:   IGNORE,
	lexlex.TokDo:       DO,
	lexlex.TokWhen:     WHEN,
	lexlex.TokName:     NAME,
}

func (p *parser) lex(v *yySymType) (int, string) {
	tok, ok := p.Next()

	if !ok {
		return 0, ""
	}

	switch tok.Kind {
	case lexlex.TokNewline:
		return NEWLINE, "a new line"
	case lexlex.TokDefine, lexlex.TokLbrace:
		p.declLine = tok.At.Line
	case lexlex.TokName:
		v.str = tok.Text
	case lexlex.TokClass:
		v.str = strings.TrimSuffix(strings.TrimPrefix(tok.Text, "[:"), ":]")
	case lexlex.TokText:
		v.str = literalOf(tok.Text)
	case lexlex.TokAnycase:
		v.str = literalOf(strings.TrimSuffix(tok.Text, "i"))
	case lexlex.TokUnterminated:
		if strings.HasSuffix(tok.Text, `\\`) {
			p.Fail("a backslash with nothing after it")
		} else {
			p.Fail("a literal runs off the end of its line")
		}

		return 0, ""
	}

	return kinds[tok.Kind], tok.Text
}

func literalOf(text string) string {
	if len(text) >= 2 && strings.HasPrefix(text, `"`) && strings.HasSuffix(text, `"`) {
		return unquote(text[1 : len(text)-1])
	}

	return unquote(text)
}

func unquote(s string) string {
	var b strings.Builder
	rs := []rune(s)

	for i := 0; i < len(rs); i++ {
		if rs[i] == '\\' && i+1 < len(rs) {
			b.WriteRune(escape(rs[i+1]))
			i++

			continue
		}

		b.WriteRune(rs[i])
	}

	return b.String()
}

func escape(r rune) rune {
	switch r {
	case 'n':
		return '\n'
	case 't':
		return '\t'
	case 'r':
		return '\r'
	case 'v':
		return '\v'
	case 'f':
		return '\f'
	}

	return r
}

func (p *parser) unexpected() string { return lexrun.Unexpected(p.tok, p.tokText) }
