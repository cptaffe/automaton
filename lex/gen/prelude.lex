digit  = [0-9]
alpha  = [a-zA-Z]
alnum  = alpha | digit
lower  = [a-z]
upper  = [A-Z]
xdigit = digit | [a-fA-F]
word   = alnum | "_"
blank  = [ \t]
space  = blank | [\n\v\f\r]
punct  = [!-/] | [:-@] | [\[-`] | [{-~]
graph  = [!-~]
print  = [ -~]

alphabetic  = unicode.l | unicode.nl | unicode.other-alphabetic
lowercase   = unicode.ll | unicode.other-lowercase
uppercase   = unicode.lu | unicode.other-uppercase

not-ident   = unicode.pattern-syntax | unicode.pattern-white-space
id-start    = (unicode.l | unicode.nl | unicode.other-id-start) \ not-ident
id-continue = (id-start | unicode.mn | unicode.mc | unicode.nd | unicode.pc | unicode.other-id-continue) \ not-ident
