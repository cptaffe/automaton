package fixed

//go:generate go run ../cmd/batch -n 100 -seed 1 -dir . -import github.com/cptaffe/automaton/internal/lexfuzz/fixed
//go:generate go run ../../cmd/lexcases -o internal/cases -import github.com/cptaffe/automaton/internal/lexfuzz/fixed/internal/cases testdata/lexers.txtar
