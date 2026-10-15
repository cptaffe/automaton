GOLANGCI = go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.14.0

.PHONY: check fuzz generate lint test

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

FUZZTIME = 1m
FUZZN = 1000
FUZZSEED := $(shell date +%s)
MODULE = github.com/cptaffe/automaton
RUN = internal/lexfuzz/run

fuzz:
	GOMEMLIMIT=512MiB go test ./fa -run '^$$' -fuzz FuzzBuildAgreesWithRegexp -fuzztime $(FUZZTIME) -parallel 2
	rm -rf $(RUN)
	go run ./internal/lexfuzz/cmd/batch -n $(FUZZN) -seed $(FUZZSEED) -dir $(RUN) -import $(MODULE)/$(RUN)
	go run ./internal/cmd/lexcases -o $(RUN)/internal/cases -import $(MODULE)/$(RUN)/internal/cases $(RUN)/testdata/lexers.txtar
	GOFLAGS=-p=2 GOMAXPROCS=2 GOMEMLIMIT=512MiB go test ./$(RUN) || { echo "make: lexers from seed $(FUZZSEED) disagree; kept in $(RUN)" >&2; exit 1; }
	rm -rf $(RUN)
	GOMEMLIMIT=512MiB go test ./seq/hotkey -run '^$$' -fuzz FuzzMachineAgreesWithAReference -fuzztime $(FUZZTIME) -parallel 2
