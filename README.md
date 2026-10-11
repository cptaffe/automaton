# automaton

Lexers and key-sequence languages, compiled to tables.

You describe a lexer, a dial plan, or a set of keyboard shortcuts in a small
language. automaton checks it for conflicts, compiles it to a table, and runs
it or draws it. The runtimes depend only on the standard library, so a
program that uses a generated table pulls in nothing else.

It started inside two projects: a payphone emulator, whose keypad menus are
trees of key sequences, and an [Acme](https://9fans.github.io/plan9port/man/man1/acme.html)
hotkey daemon, whose shortcuts are the same thing over a keyboard.

| Package | What it does |
|---|---|
| [`automaton`](.) | Rune sets, regular expressions, NFA → DFA, minimisation |
| [`lex`](lex) | The lexer runtime that generated lexers use |
| [`lex/gen`](lex/gen), [`cmd/genlex`](cmd/genlex) | The `.lex` language → lexer tables, as Go |
| [`seq`](seq) | Key-sequence grammars → tables, and a machine to step them |
| [`seq/keypad`](seq/keypad), [`seq/hotkey`](seq/hotkey), [`cmd/genseq`](cmd/genseq) | Two languages for key sequences |
| [`lr`](lr), [`cmd/lrdot`](cmd/lrdot) | goyacc's `y.output` → the grammar and its LR automaton |
| [`dot`](dot), [`lex/draw`](lex/draw), [`seq/draw`](seq/draw), [`lr/draw`](lr/draw) | Graphviz drawings of all of the above |

```sh
go install github.com/cptaffe/automaton/cmd/...@latest
```

## Lexers

A `.lex` file has scopes of rules; each rule matches the longest input it can
and emits a token, ignores it, or moves to another scope. A rule that is only
`-> scope` reads that scope's rules in where it stands. A scope that takes a
parameter returns to whoever entered it, and may enter itself, so comments can
nest:

```
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
```

```go
//go:generate go run github.com/cptaffe/automaton/cmd/genlex -go scan.gen.go -package scan comments.lex

l, err := lex.New(scan.Spec, r, "start")
for tok, ok := l.Next(); ok; tok, ok = l.Next() {
	fmt.Println(scan.Spec.Token(tok.Kind), tok.Text)
}
```

![comments.lex as a machine](docs/comments.svg)

## Key sequences

[`seq`](seq) compiles scopes of key sequences into a table, refusing two rules
on the same keys and a rule that is also the start of a longer one unless the
scope says how long to wait for the next key. A machine starts in the first
scope and returns to its scope's start after each match; `-> scope` after a
rule moves it, and `-> scope when ...` on a line of its own reads another
scope's rules in, as long as the condition holds. A key is taken only when a
rule that holds can use it. Both dialects share it.

### Keypad

A dial plan, where `N` is 2–9, `X` is 0–9, and `T` settles a number that
could still grow. A plan can span several trees: `=> menu` after a rule
carries on in the `menu` tree once the call is answered, and `keypad.Compile`
links the trees it is given into one table, naming each scope after its
tree:

```
%delay T 5s
%delay abandon 16s

start:
  %settles T
  0             do operator
  *98           do voicemail  -> mailbox
  NXX-XXXX      do local
  960-NXX-XXXX  do long-distance
  ... abandon   do dial-again

mailbox:
  1             do play       when messages
  #             do hang-up    -> start
```

```go
tree, err := keypad.Parse(f, "dial.tree")
t, err := keypad.Compile(tree)
m, err := seq.NewMachine(t, "")
taken, fired := m.Step(t.Alpha.Of("*"), seq.Env{"messages": "1"})
```

![dial.tree as a machine](docs/dial.svg)

### Hotkeys

Keyboard shortcuts, each running an rc script, with a shared prelude. `main`
reads in `acme`'s rules only while Acme has the keyboard, and `marks` is a mode
that `escape` leaves:

```
%{
fn ctl { echo $1 | 9p write acme/`{9p read acme-focused/focus}/ctl }
%}
%delay settle 1s

main:
	-> acme when app == "devdraw"

acme:
	%settles settle
	command-s		{ ctl put }	# save
	command-w		{ ctl del }	# close
	control-x control-s	{ ctl put }	# save, the emacs way
	control-x		{ ctl show }	# show, unless control-s follows
	command-m		-> marks

marks:
	m			{ ctl mark }
	escape			{ ctl show } -> main
```

```go
r, err := hotkey.Parse(f, "hotkeys", nil)
m, err := seq.NewMachine(r.Table, "")
env := seq.Env{"app": "devdraw"}
taken, fired := m.Step(r.Table.Alpha.Of("control-x"), env)
for _, a := range fired {
	exec.Command("rc", "-c", r.Prelude+"\n"+a.Action).Start()
}
for _, name := range m.Waits() {
	time.AfterFunc(r.Table.Delays[name], func() { m.Step(r.Table.Alpha.Of(name), env) })
}
```

![hotkeys as a machine](docs/hotkeys.svg)

## LR automata

`lrdot` reads the report goyacc writes with `-v` and draws the grammar and the
machine over its tokens, noting when the grammar is too nested for a machine
to read it exactly:

```sh
goyacc -o /dev/null -v y.output gram.y
lrdot -name gram y.output gram.dot
```

The drawings in [docs](docs) are made by [`docs/draw.sh`](docs/draw.sh) from
the files in [examples](examples).
