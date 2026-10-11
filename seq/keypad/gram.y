%{
package keypad

import "github.com/cptaffe/automaton/seq"
%}

%union {
	str  string
	move move
	word word
	sym  symv
	pat  pat
	cond seq.Cond
	rule seq.Rule
	any  seq.Any
}

%token <word> IDENT DELAY SETTLES
%token <sym> SYM
%token ARROW COLON DOTS DO WHEN AND OR NOT LPAREN RPAREN NEWLINE

%type <str> action param_
%type <move> goto_
%type <pat> syms
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
	| LPAREN IDENT RPAREN   { $$ = $2.text }
	;

decl:
	  DELAY IDENT IDENT NEWLINE   { yylex.(*lexer).delay($1.line, $2.text, $3.text) }
	| SETTLES IDENT NEWLINE       { yylex.(*lexer).settles($1.line, $2.text) }
	;

rule:
	  syms action goto_ when_ NEWLINE
		{
			$$ = seq.Rule{Pattern: yylex.(*lexer).text($1), Elems: $1.elems, Line: $1.line, Action: $2, Goto: $3.to, Arg: $3.arg, When: $4}
		}
	;

anyrule:
	  DOTS syms action goto_ when_ NEWLINE
		{
			$$ = seq.Any{Pattern: yylex.(*lexer).text($2), Line: $2.line, Action: $3, Goto: $4.to, Arg: $4.arg, When: $5}
			if len($2.elems) == 1 {
				$$.Elem = $2.elems[0]
			} else {
				yylex.(*lexer).FailAt($2.line, "`... %s` is more than one symbol", yylex.(*lexer).text($2))
			}
		}
	;

syms:
	  SYM           { $$ = one($1) }
	| IDENT         { $$ = one(event($1)) }
	| syms SYM      { $$ = add($1, $2) }
	| syms IDENT    { $$ = add($1, event($2)) }
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
