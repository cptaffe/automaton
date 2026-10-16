%{
package keypad

import (
	"strings"

	"github.com/cptaffe/automaton/seq"
)
%}

%union {
	str  string
	move move
	word word
	pat  pat
	cond seq.Cond
	rule seq.Rule
	any  seq.Any
}

%token <word> IDENT DELAY BAR PLUS OPT MINUS LPAREN RPAREN CALL SYM
%token ARROW COLON DOTS DO WHEN AND OR NOT NEWLINE

%type <str> action param_
%type <move> goto_
%type <pat> pattern cat piece diff atom
%type <cond> when_ expr term factor
%type <rule> rule
%type <any> anyrule

%%

file:
	  lines
	;

lines:
	| lines NEWLINE
	| lines header
	| lines decl
	| lines rule      { yylex.(*lexer).rule($2) }
	| lines anyrule   { yylex.(*lexer).anyRule($2) }
	;

header:
	  IDENT param_ COLON NEWLINE   { yylex.(*lexer).scope($1, $2) }
	;

param_:
		{ $$ = "" }
	| CALL IDENT RPAREN     { $$ = $2.text }
	;

decl:
	  DELAY IDENT IDENT NEWLINE   { yylex.(*lexer).delay($1.line, $2.text, $3.text) }
	;

rule:
	  pattern action goto_ when_ NEWLINE
		{
			$$ = seq.Rule{Pattern: strings.TrimSpace($1.text), Expr: $1.e, Line: $1.line, Action: $2, Goto: $3.to, Arg: $3.arg, When: $4}
		}
	;

anyrule:
	  DOTS pattern action goto_ when_ NEWLINE
		{
			$$ = seq.Any{Pattern: strings.TrimSpace($2.text), Elem: $2.e, Line: $2.line, Action: $3, Goto: $4.to, Arg: $4.arg, When: $5}
			if !$2.set {
				yylex.(*lexer).FailAt($2.line, "`... %s` is more than one symbol", strings.TrimSpace($2.text))
			}
		}
	;

pattern:
	  cat
	| pattern BAR cat      { $$ = alt($1, $2, $3) }
	;

cat:
	  piece
	| cat piece            { $$ = cat($1, $2) }
	;

piece:
	  diff
	| diff PLUS            { $$ = repeat($1, $2, seq.Plus{Of: $1.e}) }
	| diff OPT             { $$ = repeat($1, $2, seq.Opt{Of: $1.e}) }
	;

diff:
	  atom
	| diff MINUS atom      { $$ = minus($1, $2, $3) }
	;

atom:
	  SYM                  { $$ = symbol($1) }
	| IDENT                { $$ = name($1) }
	| LPAREN pattern RPAREN
		{
			$$ = $2
			$$.text = $1.lead + "(" + $2.text + $3.lead + ")"
		}
	;

action:
		{ $$ = "" }
	| DO IDENT      { $$ = $2.text }
	;

goto_:
		{ $$ = move{} }
	| ARROW IDENT   { $$ = move{to: $2.text} }
	| ARROW IDENT LPAREN IDENT RPAREN   { $$ = move{to: $2.text, arg: $4.text} }
	;

when_:
		{ $$ = nil }
	| WHEN expr     { $$ = $2 }
	;

expr:
	  term          { $$ = $1 }
	| expr OR term  { $$ = seq.AnyOf($1, $3) }
	;

term:
	  factor            { $$ = $1 }
	| term AND factor   { $$ = seq.AllOf($1, $3) }
	;

factor:
	  IDENT              { $$ = seq.Name($1.text) }
	| NOT factor         { $$ = seq.Not{Of: $2} }
	| LPAREN expr RPAREN { $$ = $2 }
	;
