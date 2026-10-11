ident  = alpha alnum* (("-" | ".") alnum+)*
quoted = "\"" ([^"\n] - [\\] | "\\" any)* "\""
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
  quoted "i"  do unescape  emit anycase
  quoted      do unescape  emit text

  "\"" ([^"\n] - [\\] | "\\" any)*  emit unterminated

  "includes"  emit includes  -> name
  "emit"      emit emit      -> name
  "ignore"    emit ignore
  "do"        emit do        -> name
  "when"      emit when      -> name
  ident       emit name
}

name includes start {
  ident       emit name      -> start

  "\n"        emit newline   -> start
}

klass(then) {
  "]"         emit rbrack    -> then

  "\n"        emit newline
  "^"         emit caret
  "-"         emit dash
  "[:" (alnum | "-" | ".")+ ":]"  emit class
  "\\" any    do unescape  emit text
  inner       emit text
}
