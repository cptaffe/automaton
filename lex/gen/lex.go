package gen

//go:generate go run ../../cmd/genlex -go internal/scan/scan.gen.go -package scan lex.lex

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/cptaffe/automaton/fa"
	"github.com/cptaffe/automaton/lex"
	"github.com/cptaffe/automaton/lex/gen/internal/scan"
)

type word struct {
	text string
	line int
}

type parser struct {
	*lex.Reader

	bare  bool
	named map[string]fa.Expr[rune]

	given map[string]string
	order []string

	file *File
	ctx  string

	tok     int
	tokText string
	tokLine int

	began, end int
	reading    []byte
	stmt       int
	stmtText   string
}

func newParser(ctx context.Context, r io.Reader, from string, bare bool) (*parser, error) {
	in, err := lex.Open(ctx, scan.Spec, r, "start", from)

	if err != nil {
		return nil, err
	}

	p := &parser{Reader: in, bare: bare, named: predefined(bare), given: given(bare),
		file: &File{Path: from}}

	if !bare {
		for _, name := range builtinNames(bare) {
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
	scan.TokNewline: NEWLINE,
	scan.TokArrow:   ARROW,
	scan.TokDefine:  DEFINE,
	scan.TokLbrace:  LBRACE,
	scan.TokRbrace:  RBRACE,
	scan.TokOr:      OR,
	scan.TokStar:    STAR,
	scan.TokPlus:    PLUS,
	scan.TokOpt:     OPT,
	scan.TokLparen:  LPAREN,
	scan.TokRparen:  RPAREN,
	scan.TokLbrack:  LBRACK,
	scan.TokRbrack:  RBRACK,
	scan.TokCaret:   CARET,
	scan.TokDash:    DASH,
	scan.TokMinus:   MINUS,
	scan.TokClass:   CLASS,
	scan.TokText:    TEXT,
	scan.TokAnycase: ANYCASE,
	scan.TokEmit:    EMIT,
	scan.TokIgnore:  IGNORE,
	scan.TokDo:      DO,
	scan.TokName:    NAME,
}

func (p *parser) lex(v *yySymType) (int, string) {
	tok, ok := p.Next()

	if !ok {
		return 0, ""
	}

	p.tokLine = tok.At.Line
	p.statement(tok)

	switch tok.Kind {
	case scan.TokNewline:
		return NEWLINE, "a new line"
	case scan.TokName:
		v.word = word{tok.Text, tok.At.Line}
	case scan.TokClass:
		v.str = strings.TrimSuffix(strings.TrimPrefix(tok.Text, "[:"), ":]")
	case scan.TokText:
		v.str = literalOf(tok.Text)
	case scan.TokAnycase:
		v.str = literalOf(strings.TrimSuffix(tok.Text, "i"))
	case scan.TokUnterminated:
		if trailing := len(tok.Text) - len(strings.TrimRight(tok.Text, `\`)); trailing%2 == 1 {
			p.Fail("a backslash with nothing after it")
		} else {
			p.Fail("a literal runs off the end of its line")
		}

		return 0, ""
	}

	return kinds[tok.Kind], tok.Text
}

func (p *parser) statement(tok lex.Token) {
	switch {
	case tok.Kind == scan.TokNewline && len(p.reading) > 0:
		p.stmt, p.stmtText, p.reading = p.began, string(p.reading), p.reading[:0]
	case tok.Kind == scan.TokNewline:
	case len(p.reading) == 0:
		p.began = tok.At.Line
		p.reading = append(p.reading, tok.Text...)
	case tok.At.Offset > p.end:
		p.reading = append(p.reading, ' ')
		fallthrough
	default:
		p.reading = append(p.reading, tok.Text...)
	}

	p.end = tok.End()
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

func (p *parser) unexpected() string {
	if p.tok == 0 {
		return "the file ends in the middle of a rule"
	}

	return fmt.Sprintf("%s cannot stand here", p.tokText)
}

func (p *parser) blame() int {
	if p.tok == 0 {
		return p.Line()
	}

	return p.tokLine
}
