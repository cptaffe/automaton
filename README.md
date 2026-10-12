# automaton

Small languages for lexers, dial plans and keyboard shortcuts, compiled to
tables you can run, check and draw.

Each language is checked for conflicts when it compiles, so a rule that can
never fire, or two rules on the same input, is an error rather than a
surprise. The runtimes use only the standard library.

```sh
go install github.com/cptaffe/automaton/cmd/...@latest
```

## Packages

| Package | |
|---|---|
| [`fa`](fa) | Finite automata over any integer symbol: sets, expressions, NFA → DFA, minimisation |
| [`lex`](lex) | The runtime generated lexers use |
| [`lex/gen`](lex/gen) | The `.lex` language, compiled to lexer tables |
| [`seq`](seq) | Key-sequence grammars, compiled to tables, and a machine to run them |
| [`seq/keypad`](seq/keypad) | Dial plans for a payphone keypad |
| [`seq/hotkey`](seq/hotkey) | Keyboard shortcuts that run rc scripts |
| [`lr`](lr) | The grammar in goyacc's `y.output`, and its machine over tokens |
| [`dot`](dot) | Graphviz drawings; `lex/draw`, `seq/draw` and `lr/draw` adapt to it |
| [`cmd/genlex`](cmd/genlex) | `.lex` → Go tables and drawings |
| [`cmd/genseq`](cmd/genseq) | Dial plans and shortcuts → drawings |
| [`cmd/lrdot`](cmd/lrdot) | `y.output` → drawings |

## Lexers

A `.lex` file is a set of scopes. Each rule reads the longest input it can,
then emits a token, ignores the input, or moves to another scope. A scope with
a parameter returns to wherever it was entered from, so comments can nest:

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
if err != nil {
	return err
}
for tok, ok := l.Next(); ok; tok, ok = l.Next() {
	fmt.Println(scan.Spec.Token(tok.Kind), tok.Text)
}
return l.Err()
```

![comments.lex](docs/comments.svg)

## Key sequences

Both dialects compile to the same kind of table:

- A machine starts in the first scope, and after each match returns to the
  start of the scope it is in.
- `-> scope` after a rule moves to that scope.
- `-> scope` on a line of its own reads that scope's rules in, and
  `-> scope when ...` does so only while the condition holds. A key is taken
  only when a rule that holds can use it.
- A rule that is also the start of a longer one is a conflict, unless the
  scope says with `%settles` how long to wait for the next key.
- A scope can take a parameter, `dialing(then)`, and `this` names the scope a
  rule is in.

### Keypad

`N` is 2–9 and `X` is 0–9. `dialing(then)` waits for the call to be answered,
and `connected`, with no rules, ends the plan:

```
%delay T 5s
%delay abandon 16s

start:
  %settles T
  0             do operator         -> connected
  *98           do voicemail        -> dialing(mailbox)
  NXX-XXXX      do local            -> connected
  960-NXX-XXXX  do long-distance    -> connected
  ... abandon   do dial-again

dialing(then):
  answered      -> then

connected:

mailbox:
  1             do play       when messages
  #             do hang-up    -> start
```

```go
plan, err := keypad.Parse(f, "dial.tree")
if err != nil {
	return err
}
t, err := keypad.Compile(plan)
if err != nil {
	return err
}
m, err := seq.NewMachine(t, "")
if err != nil {
	return err
}
taken, fired := m.Step(t.Alpha.Of("*"), seq.Env{"messages": "1"})
```

Several plans compile together with `keypad.Compile`, and a `->` to another
plan's name goes to its first scope.

![dial.tree](docs/dial.svg)

### Hotkeys

Shortcuts run rc scripts that share a prelude. `main` reads in `acme`'s rules
only while Acme has the keyboard, and `marks` is a mode that `escape` leaves:

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
if err != nil {
	return err
}
m, err := seq.NewMachine(r.Table, "")
if err != nil {
	return err
}
taken, fired := m.Step(r.Table.Alpha.Of("control-x"), seq.Env{"app": "devdraw"})
for _, a := range fired {
	exec.Command("rc", "-c", r.Prelude+"\n"+a.Action).Start()
}
```

`m.Waits()` names the delays to start a timer for. When one runs out, step
the machine with it.

![hotkeys](docs/hotkeys.svg)

## LR automata

`lrdot` reads the report goyacc writes with `-v`, and draws the grammar and
the machine over its tokens. It notes when the grammar nests too deeply for a
machine to read it exactly:

```sh
goyacc -o /dev/null -v y.output gram.y
lrdot -name gram y.output gram.dot
```

The drawings are made by [`docs/draw.sh`](docs/draw.sh) from
[`examples`](examples).
