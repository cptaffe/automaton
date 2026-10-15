#!/bin/sh
set -e
cd "$(dirname "$0")/.."
go run ./cmd/genlex -dot docs examples/comments.lex
go run ./cmd/genseq -dot docs examples/dial.tree examples/hotkeys
dark='<style>@media (prefers-color-scheme: dark) {
#graph0 > text[fill="#343a40"] { fill: #dee2e6 }
.edge [stroke="#495057"] { stroke: #adb5bd }
.edge [fill="#495057"] { fill: #adb5bd }
.cluster > text[fill="#495057"] { fill: #adb5bd }
}</style>'
for f in docs/*.dot; do
	dot -Tsvg -Gbgcolor=transparent "$f" |
		DARK="$dark" perl -pe 's/^(<g id="graph0")/$ENV{DARK}\n$1/' >"${f%.dot}.svg"
done
rm docs/*.dot docs/*-shape.svg
