%{
package gen

import "github.com/cptaffe/automaton"
%}

%union {
	word word
	str  string
	set  automaton.Set
	expr automaton.Expr
	rule Rule
	acts Acts
}

%token <word> NAME
%token <str> TEXT ANYCASE CLASS
%token DEFINE EMIT IGNORE ARROW DO LBRACE RBRACE
%token OR STAR PLUS OPT LPAREN RPAREN LBRACK RBRACK CARET DASH MINUS NEWLINE

%type <expr> expr term piece diff atom
%type <set> klass items item
%type <rule> rule
%type <acts> acts act
%type <str> param_ arg_

%%

file:
	  lines
	;

lines:
	| lines NEWLINE
	| lines define
	| lines scope
	;

scope:
	  header LBRACE NEWLINE rules RBRACE NEWLINE
	;

rules:
	| rules NEWLINE
	| rules rule   { yylex.(*parser).rule($2) }
	;

define:
	  NAME DEFINE expr NEWLINE   { yylex.(*parser).define($1, $3) }
	;

header:
	  NAME param_   { yylex.(*parser).scope($1, $2) }
	;

param_:
	  { $$ = "" }
	| LPAREN NAME RPAREN { $$ = $2.text }
	;

rule:
	  expr acts NEWLINE  { $$ = Rule{Expr: $1, Acts: $2} }
	| ARROW NAME NEWLINE { $$ = Rule{Include: $2.text} }
	;

acts:
	  act                { $$ = $1 }
	| acts act           { $$ = $1.and($2) }
	;

act:
	  EMIT NAME          { $$ = Acts{Emit: $2.text} }
	| IGNORE             { $$ = Acts{Drop: true} }
	| DO NAME            { $$ = Acts{Do: $2.text} }
	| ARROW NAME arg_    { $$ = Acts{Goto: $2.text, Arg: $3} }
	;

arg_:
	  { $$ = "" }
	| LPAREN NAME RPAREN { $$ = $2.text }
	;

expr:
	  term               { $$ = $1 }
	| expr OR term       { $$ = automaton.Alt($1, $3) }
	;

term:
	  piece              { $$ = $1 }
	| term piece         { $$ = automaton.Cat($1, $2) }
	;

piece:
	  diff               { $$ = $1 }
	| diff STAR          { $$ = automaton.Star($1) }
	| diff PLUS          { $$ = automaton.Plus($1) }
	| diff OPT           { $$ = automaton.Opt($1) }
	;

diff:
	  atom               { $$ = $1 }
	| diff MINUS atom    { $$ = yylex.(*parser).without($1, $3) }
	;

atom:
	  TEXT               { $$ = yylex.(*parser).text($1) }
	| ANYCASE            { $$ = yylex.(*parser).anycase($1) }
	| NAME               { $$ = yylex.(*parser).use($1) }
	| klass              { $$ = automaton.Lit($1) }
	| LPAREN expr RPAREN { $$ = $2 }
	;

klass:
	  LBRACK items RBRACK             { $$ = $2 }
	| LBRACK CARET items RBRACK       { $$ = $3.Negate() }
	;

items:
	  item                { $$ = $1 }
	| items item          { $$ = $1.Union($2) }
	;

item:
	  TEXT                { $$ = yylex.(*parser).runes($1) }
	| TEXT DASH TEXT      { $$ = yylex.(*parser).span($1, $3) }
	| CLASS               { $$ = yylex.(*parser).category($1) }
	;
