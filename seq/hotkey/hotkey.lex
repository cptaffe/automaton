part   = (alphabetic | digit | [_.])+
mark   = any - alphabetic - digit - blank - [\n"{}#:]
chord  = part ("-" (part | mark))*
ident  = (alphabetic | "_") (alphabetic | digit | "_")*
quoted = "\"" [^"\n]* "\"" | "'" [^'\n]* "'"

start {
  "#" [^\n]*          emit comment
  blank+              ignore
  "\n"                emit newline
  "%{" blank* "\n"    emit open    -> prelude
  "%delay"            emit delay
  "..."               emit dots
  "->"                emit arrow
  ":"                 emit colon
  "{"                 emit lbrace  -> block(this)
  "when"              emit when    -> cond
  "|"                 emit bar
  "("                 emit lparen
  ")"                 emit rparen
  "*"                 emit star
  "+"                 emit plus
  "?"                 emit opt
  "/"                 emit slash
  "-"                 emit minus
  "\"" [^"\n]+ "\""    emit key
  chord               emit word
}

cond {
  "#" [^\n]*          emit comment
  blank+              ignore
  "\n"                emit newline -> start
  "=="                emit eq
  "!="                emit ne
  "~"                 emit match
  "&&"                emit and
  "||"                emit or
  "!"                 emit not
  "("                 emit lparen
  ")"                 emit rparen
  quoted              emit string
  ident               emit ident
}

prelude {
  "%}" blank* "\n"?   emit close   -> start
  [^\n]* "\n"         emit text
}

block(then) {
  "{"                 emit lbrace  -> block(this)
  "}"                 emit rbrace  -> then
  "'" [^']* "'"       emit text
  "#" [^\n]*          emit text
  [^{}'#]+            emit text
}
