%{
package hotkey

import "github.com/cptaffe/automaton/seq"
%}

%union {
	word word
	pat  pat
	str  string
	cond seq.Cond
	rule rule
}

%token <word> WORD STRING IDENT DELAY SETTLES DOTS LBRACE RBRACE OPEN CLOSE
%token NEWLINE COLON ARROW WHEN EQ NE MATCH AND OR NOT LPAREN RPAREN TEXT

%type <pat> keys
%type <str> action goto_ block
%type <cond> when_ expr term factor
%type <rule> rule

%%

file:
	  lines
	;

lines:
	| lines NEWLINE
	| lines OPEN text CLOSE   { yylex.(*lexer).prelude($2, $4) }
	| lines header
	| lines decl
	| lines rule              { yylex.(*lexer).rule($2) }
	| lines DOTS rule         { yylex.(*lexer).anyRule($3) }
	;

text:
	| text TEXT
	;

header:
	  WORD COLON when_ NEWLINE   { yylex.(*lexer).scope($1, $3) }
	;

when_:
		{ $$ = nil }
	| WHEN expr   { $$ = $2 }
	;

decl:
	  DELAY WORD WORD NEWLINE   { yylex.(*lexer).delay($1.line, $2.text, $3.text) }
	| SETTLES WORD NEWLINE      { yylex.(*lexer).settles($1.line, $2.text) }
	;

rule:
	  keys action goto_ NEWLINE   { $$ = rule{$1, $2, $3} }
	;

keys:
	  WORD        { $$ = pat{words: []word{$1}, line: $1.line} }
	| keys WORD   { $$ = pat{words: append($1.words, $2), line: $1.line} }
	;

action:
		{ $$ = "" }
	| block       { $$ = $1 }
	;

block:
	  LBRACE body RBRACE   { $$ = yylex.(*lexer).between($1, $3) }
	;

body:
	| body TEXT
	| body block
	;

goto_:
		{ $$ = "" }
	| ARROW WORD  { $$ = $2.text }
	;

expr:
	  term            { $$ = $1 }
	| expr OR term    { $$ = or($1, $3) }
	;

term:
	  factor            { $$ = $1 }
	| term AND factor   { $$ = and($1, $3) }
	;

factor:
	  NOT factor           { $$ = seq.Not{Of: $2} }
	| LPAREN expr RPAREN   { $$ = $2 }
	| IDENT                { $$ = seq.Name($1.text) }
	| IDENT EQ STRING      { $$ = yylex.(*lexer).compare($1, "==", $3) }
	| IDENT NE STRING      { $$ = yylex.(*lexer).compare($1, "!=", $3) }
	| IDENT MATCH STRING   { $$ = yylex.(*lexer).compare($1, "~", $3) }
	;
