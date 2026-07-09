// staticlint — multichecker for the metrics project.
//
// # Overview
//
// staticlint is a composite static-analysis tool that runs a curated set
// of analyzers in a single pass over a Go package. It is built with
// [golang.org/x/tools/go/analysis/multichecker], which orchestrates
// all analyzers, shares type-checking results between them, and merges
// their diagnostics into a unified output.
//
// # Running the tool
//
//	go run ./cmd/staticlint ./...          # analyze whole project
//	go run ./cmd/staticlint ./cmd/server/  # analyze one package
//
// Or build once and reuse:
//
//	go build -o staticlint ./cmd/staticlint
//	./staticlint ./...
//
// Pass -help to list every available analyzer and its description:
//
//	go run ./cmd/staticlint -help
//
// # Included analyzers
//
// ## 1. Standard passes — golang.org/x/tools/go/analysis/passes
//
// These are the same analyzers that back go vet and the official Go
// toolchain. Each reports a specific class of likely programming mistakes.
//
//   - appends      — detect missing values passed to append
//   - asmdecl      — check assembly declarations against Go signatures
//   - assign       — detect useless assignments
//   - atomic       — check for non-atomic 64-bit accesses on 32-bit targets
//   - atomicalign  — check for non-64-bit-aligned atomic operations
//   - bools        — detect mistakes with boolean operators
//   - buildtag     — verify //go:build and +build directives
//   - cgocall      — detect illegal calls into C code
//   - composite    — check composite literals for unkeyed fields
//   - copylock     — detect copies of sync.Locker values
//   - deepequalerrors — warn when reflect.DeepEqual compares errors
//   - defers       — detect common mistakes with defer statements
//   - directive    — check Go toolchain directives (//go:...)
//   - errorsas     — check that errors.As target is a non-nil pointer
//   - fieldalignment — suggest reordering struct fields to save memory
//   - httpresponse — detect mistakes when using net/http response bodies
//   - ifaceassert  — detect impossible interface-to-interface type assertions
//   - loopclosure  — detect references to loop variables in closures
//   - lostcancel   — detect failure to call a context cancel function
//   - nilfunc      — detect useless comparisons of functions to nil
//   - nilness      — check for redundant or impossible nil comparisons
//   - printf       — verify format string / argument consistency
//   - reflectvaluecompare — detect equality comparisons of reflect.Value
//   - shadow       — check for variable shadowing
//   - shift        — detect bit shifts that exceed the width of the type
//   - sigchanyzer  — detect misuse of os/signal.Notify
//   - slog         — check consistency of slog calls
//   - sortslice    — detect calls to sort.Slice that don't swap correctly
//   - stdmethods   — verify signature conventions for well-known interfaces
//   - stdversion   — warn about uses of symbols newer than the go.mod version
//   - stringintconv — detect conversions from int to string
//   - structtag    — check struct field tags for well-formed syntax
//   - testinggoroutine — detect t.Fatal inside goroutines spawned by tests
//   - tests        — detect common test function naming mistakes
//   - timeformat   — detect wrong format strings for time.Time.Format
//   - unmarshal    — detect passing non-pointer values to Unmarshal
//   - unreachable  — detect unreachable code
//   - unsafeptr    — detect invalid conversions from uintptr to unsafe.Pointer
//   - unusedresult — detect unused results of calls to certain functions
//   - unusedwrite  — detect unused writes to struct fields or array elements
//   - waitgroup    — detect misuse of sync.WaitGroup
//
// ## 2. SA class — staticcheck.io (honnef.co/go/tools/staticcheck)
//
// All SA-class analyzers from the staticcheck suite. These check for bugs
// and incorrect usage of the standard library:
//
//   - SA1xxx — misuse of the standard library (incorrect API usage)
//   - SA2xxx — concurrency issues (e.g. incorrect use of sync primitives)
//   - SA3xxx — testing-related issues
//   - SA4xxx — code simplifications where code is provably dead or useless
//   - SA5xxx — correctness issues (panics, infinite loops, etc.)
//   - SA6xxx — performance issues
//   - SA9xxx — dubious code constructs
//
// ## 3. S class — staticcheck.io simple (honnef.co/go/tools/simple)
//
// One additional non-SA class from staticcheck: analyzers that suggest
// idiomatic, simpler Go code without changing behavior:
//
//   - S1xxx — code simplifications (use of builtins, range, etc.)
//
// ## 4. Public analyzers
//
//   - errcheck (github.com/kisielk/errcheck/errcheck)
//     Reports calls whose error return value is silently discarded.
//     Unchecked errors are one of the most common sources of silent bugs
//     in Go programs.
//
//   - bodyclose (github.com/timakin/bodyclose/passes/bodyclose)
//     Reports HTTP response bodies that are not properly closed.
//     Forgetting to close resp.Body causes goroutine and connection leaks.
//
// ## 5. Custom analyzer — noexit (metrics/cmd/staticlint/noexit)
//
// Prohibits direct calls to os.Exit inside the main function of the main
// package. See [metrics/cmd/staticlint/noexit] for the full rationale.
package main

import (
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/multichecker"
	"golang.org/x/tools/go/analysis/passes/appends"
	"golang.org/x/tools/go/analysis/passes/asmdecl"
	"golang.org/x/tools/go/analysis/passes/assign"
	"golang.org/x/tools/go/analysis/passes/atomic"
	"golang.org/x/tools/go/analysis/passes/atomicalign"
	"golang.org/x/tools/go/analysis/passes/bools"
	"golang.org/x/tools/go/analysis/passes/buildtag"
	"golang.org/x/tools/go/analysis/passes/cgocall"
	"golang.org/x/tools/go/analysis/passes/composite"
	"golang.org/x/tools/go/analysis/passes/copylock"
	"golang.org/x/tools/go/analysis/passes/deepequalerrors"
	"golang.org/x/tools/go/analysis/passes/defers"
	"golang.org/x/tools/go/analysis/passes/directive"
	"golang.org/x/tools/go/analysis/passes/errorsas"
	"golang.org/x/tools/go/analysis/passes/fieldalignment"
	"golang.org/x/tools/go/analysis/passes/httpresponse"
	"golang.org/x/tools/go/analysis/passes/ifaceassert"
	"golang.org/x/tools/go/analysis/passes/loopclosure"
	"golang.org/x/tools/go/analysis/passes/lostcancel"
	"golang.org/x/tools/go/analysis/passes/nilfunc"
	"golang.org/x/tools/go/analysis/passes/nilness"
	"golang.org/x/tools/go/analysis/passes/printf"
	"golang.org/x/tools/go/analysis/passes/reflectvaluecompare"
	"golang.org/x/tools/go/analysis/passes/shadow"
	"golang.org/x/tools/go/analysis/passes/shift"
	"golang.org/x/tools/go/analysis/passes/sigchanyzer"
	"golang.org/x/tools/go/analysis/passes/slog"
	"golang.org/x/tools/go/analysis/passes/sortslice"
	"golang.org/x/tools/go/analysis/passes/stdmethods"
	"golang.org/x/tools/go/analysis/passes/stdversion"
	"golang.org/x/tools/go/analysis/passes/stringintconv"
	"golang.org/x/tools/go/analysis/passes/structtag"
	"golang.org/x/tools/go/analysis/passes/testinggoroutine"
	"golang.org/x/tools/go/analysis/passes/tests"
	"golang.org/x/tools/go/analysis/passes/timeformat"
	"golang.org/x/tools/go/analysis/passes/unmarshal"
	"golang.org/x/tools/go/analysis/passes/unreachable"
	"golang.org/x/tools/go/analysis/passes/unsafeptr"
	"golang.org/x/tools/go/analysis/passes/unusedresult"
	"golang.org/x/tools/go/analysis/passes/unusedwrite"
	"golang.org/x/tools/go/analysis/passes/waitgroup"

	"github.com/kisielk/errcheck/errcheck"
	"github.com/timakin/bodyclose/passes/bodyclose"

	"honnef.co/go/tools/analysis/lint"
	"honnef.co/go/tools/simple"
	"honnef.co/go/tools/staticcheck"

	"metrics/cmd/staticlint/noexit"
)

func main() {
	analyzers := []*analysis.Analyzer{
		// Standard passes.
		appends.Analyzer,
		asmdecl.Analyzer,
		assign.Analyzer,
		atomic.Analyzer,
		atomicalign.Analyzer,
		bools.Analyzer,
		buildtag.Analyzer,
		cgocall.Analyzer,
		composite.Analyzer,
		copylock.Analyzer,
		deepequalerrors.Analyzer,
		defers.Analyzer,
		directive.Analyzer,
		errorsas.Analyzer,
		fieldalignment.Analyzer,
		httpresponse.Analyzer,
		ifaceassert.Analyzer,
		loopclosure.Analyzer,
		lostcancel.Analyzer,
		nilfunc.Analyzer,
		nilness.Analyzer,
		printf.Analyzer,
		reflectvaluecompare.Analyzer,
		shadow.Analyzer,
		shift.Analyzer,
		sigchanyzer.Analyzer,
		slog.Analyzer,
		sortslice.Analyzer,
		stdmethods.Analyzer,
		stdversion.Analyzer,
		stringintconv.Analyzer,
		structtag.Analyzer,
		testinggoroutine.Analyzer,
		tests.Analyzer,
		timeformat.Analyzer,
		unmarshal.Analyzer,
		unreachable.Analyzer,
		unsafeptr.Analyzer,
		unusedresult.Analyzer,
		unusedwrite.Analyzer,
		waitgroup.Analyzer,

		// Public analyzers.
		errcheck.Analyzer,
		bodyclose.Analyzer,

		// Custom analyzer.
		noexit.Analyzer,
	}

	// SA class — all analyzers from staticcheck.io staticcheck package.
	for _, a := range staticcheck.Analyzers {
		analyzers = append(analyzers, a.Analyzer)
	}

	// S class — simplification analyzers from staticcheck.io simple package.
	analyzers = append(analyzers, lintAnalyzers(simple.Analyzers)...)

	multichecker.Main(analyzers...)
}

// lintAnalyzers extracts the underlying [analysis.Analyzer] from each
// [lint.Analyzer] wrapper used by the staticcheck suite.
func lintAnalyzers(la []*lint.Analyzer) []*analysis.Analyzer {
	result := make([]*analysis.Analyzer, len(la))
	for i, a := range la {
		result[i] = a.Analyzer
	}
	return result
}
