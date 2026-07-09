package noexit_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"metrics/cmd/staticlint/noexit"
)

// TestBadPackage verifies that os.Exit calls in main() are reported.
func TestBadPackage(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, noexit.Analyzer, "bad")
}

// TestGoodPackage verifies that main() without os.Exit produces no diagnostics.
func TestGoodPackage(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, noexit.Analyzer, "good")
}

// TestNotMainPackage verifies that os.Exit in a non-main package is not reported.
func TestNotMainPackage(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, noexit.Analyzer, "notmain")
}
