name   = (alpha | "_") (alnum | "_" | "." | "-")*
double = "\"" ([^"] \ [\\] | "\\" any)* "\""?
single = "'" ([^'] \ [\\] | "\\" any)* "'"?
klass  = "[" ("[:" (alnum | "-" | ".")+ ":]" | [^\]] \ [\\] | "\\" any)* "]"?

start {
  (blank | "\n")+           ignore
  double | single | klass   emit literal
  name                      emit name
  any                       emit other
}
