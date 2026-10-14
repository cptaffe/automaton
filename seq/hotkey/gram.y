%{
package hotkey

import (
	"fmt"
	"strings"

	"github.com/cptaffe/automaton/seq"
)
%}

%union {
	word word
	pat  pat
	str  string
	cond seq.Cond
	rule rule
}

%token <word> WORD KEY STRING IDENT DELAY DOTS
%token <str> TEXT
%token NEWLINE COLON ARROW WHEN BAR STAR PLUS OPT SLASH MINUS EQ NE MATCH AND OR NOT LPAREN RPAREN LBRACE RBRACE OPEN CLOSE

%type <pat> pattern alt cat piece diff atom
%type <str> goto_ block body text
%type <cond> when_ expr term factor
%type <rule> rule

%%

file:
	  lines
	;

lines:
	| lines NEWLINE
	| lines OPEN text CLOSE   { yylex.(*lexer).addPrelude($3) }
	| lines header
	| lines decl
	| lines rule              { yylex.(*lexer).rule($2) }
	| lines ARROW WORD when_ NEWLINE   { yylex.(*lexer).include($3, $4) }
	| lines DOTS rule         { yylex.(*lexer).anyRule($3) }
	;

text:
		{ $$ = "" }
	| text TEXT   { $$ = $1 + $2 }
	;

header:
	  WORD COLON NEWLINE   { yylex.(*lexer).scope($1) }
	;

when_:
		{ $$ = nil }
	| WHEN expr   { $$ = $2 }
	;

decl:
	  DELAY WORD WORD NEWLINE         { yylex.(*lexer).delay($1.line, $2.text, $3.text) }
	;

rule:
	  pattern goto_ NEWLINE         { $$ = rule{keys: $1, to: $2} }
	| pattern block goto_ NEWLINE   { $$ = rule{keys: $1, action: $2, block: true, to: $3} }
	;

pattern:
	  alt
	| alt SLASH diff   { $$ = peek($1, $3) }
	;

alt:
	  cat
	| alt BAR cat      { $$ = join('|', " | ", $1, $3) }
	;

cat:
	  piece
	| cat piece        { $$ = join('.', " ", $1, $2) }
	;

piece:
	  diff
	| diff STAR        { $$ = after('*', $1) }
	| diff PLUS        { $$ = after('+', $1) }
	| diff OPT         { $$ = after('?', $1) }
	;

diff:
	  atom
	| diff MINUS atom  { $$ = join('-', " \\ ", $1, $3) }
	;

atom:
	  WORD                { $$ = leaf($1, $1.text) }
	| KEY                 { $$ = leaf($1, fmt.Sprintf("%q", $1.text)) }
	| LPAREN alt RPAREN   { $$ = group($2) }
	;

block:
	  LBRACE body RBRACE   { $$ = strings.TrimSpace($2) }
	;

body:
		{ $$ = "" }
	| body TEXT                 { $$ = $1 + $2 }
	| body LBRACE body RBRACE   { $$ = $1 + "{" + $3 + "}" }
	;

goto_:
		{ $$ = "" }
	| ARROW WORD  { $$ = $2.text }
	;

expr:
	  term            { $$ = $1 }
	| expr OR term    { $$ = seq.AnyOf($1, $3) }
	;

term:
	  factor            { $$ = $1 }
	| term AND factor   { $$ = seq.AllOf($1, $3) }
	;

factor:
	  NOT factor           { $$ = seq.Not{Of: $2} }
	| LPAREN expr RPAREN   { $$ = $2 }
	| IDENT                { $$ = seq.Name($1.text) }
	| IDENT EQ STRING      { $$ = yylex.(*lexer).compare($1, seq.Eq, $3) }
	| IDENT NE STRING      { $$ = yylex.(*lexer).compare($1, seq.Ne, $3) }
	| IDENT MATCH STRING   { $$ = yylex.(*lexer).compare($1, seq.Match, $3) }
	;
