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
	go test -race ./...
