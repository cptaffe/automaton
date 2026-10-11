#!/bin/sh
set -e
cd "$(dirname "$0")/.."
go run ./cmd/genlex -dot docs examples/comments.lex
go run ./cmd/genseq -dot docs examples/dial.tree examples/hotkeys
for f in docs/*.dot; do
	dot -Tsvg "$f" -o "${f%.dot}.svg"
done
rm docs/*.dot docs/*-shape.svg
