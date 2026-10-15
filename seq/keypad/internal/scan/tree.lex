key    = [0-9*#]
class  = [NX]
inner  = alphabetic | digit | "-"
plain  = alphabetic \ [NX]
ident  = inner* plain inner*

start {
  ";" [^\n]*  emit comment
  blank+      ignore
  "\n"        emit newline

  "..."       emit dots
  "->"        emit arrow

  "%delay"    emit delay

  "%" ident   emit declaration

  "do"        emit do
  "when"      emit when
  "and"       emit and
  "or"        emit or
  "not"       emit not

  ":"         emit colon
  "("         emit lparen
  ")"         emit rparen

  "|"         emit bar
  "+"         emit plus
  "?"         emit opt
  "\\"        emit minus

  key | class  emit sym
  ident       emit name
}
