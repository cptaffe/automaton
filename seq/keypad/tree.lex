key    = [0-9*#]
class  = [NX]
inner  = alphabetic | digit | "-"
plain  = alphabetic - [NX]
ident  = inner* plain inner*

start {
  ";" [^\n]*  emit comment
  blank+      emit blank
  "\n"        emit newline

  "..."       emit dots
  "->"        emit arrow
  "=>"        emit handoff

  "%delay"    emit delay
  "%settles"  emit settles

  "%" ident   emit declaration

  "do"        emit do
  "when"      emit when
  "and"       emit and
  "or"        emit or
  "not"       emit not

  ":"         emit colon
  "("         emit lparen
  ")"         emit rparen

  "-"* (key | class) "+"?  do symbol  emit sym
  "-"+        emit dash
  ident       do name    emit name
}
