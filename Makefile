GOLANGCI = go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.14.0

.PHONY: check generate lint test

check: generate lint test

generate:
	go generate ./...
	git diff --exit-code

lint:
	go vet ./...
	$(GOLANGCI) run ./...

test:
	@command -v dot >/dev/null || { echo "make: Graphviz's dot is needed for the drawing tests" >&2; exit 1; }
	go test -race ./...
