ident  = alpha alnum* (("-" | ".") alnum+)*
quoted = "\"" ([^"\n] - [\\] | "\\" [^\n])* "\""
inner  = any - [\]\^\-\n]

start {
  ";" [^\n]*  ignore
  blank+      ignore
  "\n"        emit newline

  "->"        emit arrow     -> name
  "="         emit define
  "{"         emit lbrace
  "}"         emit rbrace
  "|"         emit or
  "*"         emit star
  "+"         emit plus
  "?"         emit opt
  ")"         emit rparen
  "-"         emit minus
  "("         emit lparen    -> name
  "["         emit lbrack    -> klass(start)
  quoted "i"  emit anycase
  quoted      emit text

  "\"" ([^"\n] - [\\] | "\\" [^\n])* "\\"?  emit unterminated

  "emit"      emit emit      -> name
  "ignore"    emit ignore
  "do"        emit do        -> name
  ident       emit name
}

name {
  ident       emit name      -> start
  "\n"        emit newline   -> start
  -> start
}

klass(then) {
  "]"         emit rbrack    -> then

  "\n"        emit newline
  "^"         emit caret
  "-"         emit dash
  "[:" (alnum | "-" | ".")+ ":]"  emit class
  "\\" any    emit text
  inner       emit text
}
