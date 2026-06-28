// Package noexit provides an analyzer that prohibits direct calls to
// os.Exit in the main function of the main package.
//
// # Rationale
//
// Calling os.Exit directly inside main() bypasses deferred functions,
// preventing proper cleanup (closing files, flushing buffers, running
// graceful-shutdown logic). The idiomatic Go approach is to let main()
// return normally after all cleanup is done, or use log.Fatal only at
// the very top of the call stack where no defers have been registered.
//
// # What is checked
//
// The analyzer walks the AST of every file in the package. When the
// package name is "main", it finds the top-level function named "main"
// (with no receiver) and inspects all call expressions inside it. Any
// call whose callee resolves — via type information — to os.Exit is
// reported as a diagnostic.
//
// # False positives
//
// There are intentionally none: the check uses types.PkgName to verify
// that the qualifier "os" refers to the standard-library "os" package,
// not a user-defined identifier with the same name.
//
// # Example
//
//	// BAD: deferred cleanup is skipped
//	func main() {
//	    defer cleanup()
//	    if err := run(); err != nil {
//	        os.Exit(1) // flagged
//	    }
//	}
//
//	// GOOD: return from main, let runtime call cleanup hooks
//	func main() {
//	    defer cleanup()
//	    if err := run(); err != nil {
//	        log.Fatal(err)
//	    }
//	}
package noexit

import (
	"go/ast"
	"go/token"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// Analyzer is the entry point for the noexit static-analysis pass.
var Analyzer = &analysis.Analyzer{
	Name:     "noexit",
	Doc:      "prohibits direct calls to os.Exit inside the main function of the main package",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

// isGenerated reports whether a Go source file was auto-generated.
// The Go convention for generated files is a comment of the form:
//
//	// Code generated ... DO NOT EDIT.
//
// See https://pkg.go.dev/cmd/go#hdr-Generate_Go_files_by_processing_source
func isGenerated(fset *token.FileSet, f *ast.File) bool {
	for _, cg := range f.Comments {
		for _, c := range cg.List {
			if strings.Contains(c.Text, "DO NOT EDIT") {
				return true
			}
		}
	}
	return false
}

func run(pass *analysis.Pass) (any, error) {
	if pass.Pkg.Name() != "main" {
		return nil, nil
	}

	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	// Find the top-level func main() declaration.
	// Skip auto-generated files (e.g. go test's _testmain.go).
	var mainFn *ast.FuncDecl
	fileOf := make(map[*ast.FuncDecl]*ast.File)
	for _, f := range pass.Files {
		if isGenerated(pass.Fset, f) {
			continue
		}
		for _, decl := range f.Decls {
			fd, ok := decl.(*ast.FuncDecl)
			if ok && fd.Recv == nil && fd.Name.Name == "main" {
				mainFn = fd
				fileOf[fd] = f
			}
		}
	}
	_ = insp // inspector used for future extensions
	if mainFn == nil {
		return nil, nil
	}

	// Walk the body of main() looking for os.Exit calls.
	ast.Inspect(mainFn.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		ident, ok := sel.X.(*ast.Ident)
		if !ok {
			return true
		}
		// Use type information to ensure the qualifier refers to the
		// standard-library "os" package and not a local shadow.
		pkgName, ok := pass.TypesInfo.Uses[ident].(*types.PkgName)
		if !ok || pkgName.Imported().Path() != "os" {
			return true
		}
		if sel.Sel.Name == "Exit" {
			pass.Reportf(call.Pos(), "direct call to os.Exit in main function of main package is forbidden; use log.Fatal or let main return")
		}
		return true
	})

	return nil, nil
}
