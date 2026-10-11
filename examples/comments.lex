start {
  "/*"    emit open  -> comment(this)
  word+   emit word
  blank+  ignore
  "\n"    emit newline
}

comment(then) {
  "/*"    emit open  -> comment(this)
  "*/"    emit close -> then
  any     ignore
}
