// Application to perform a static check for code.
//
// # To run the app follow the steps:
//
// 1. In project folder make: go build ./cmd/staticlint/.
//
// 2. Run the app:  go vet -vettool=staticlint ./...
//
// # Used Analyzers
//
// • asmdecl.Analyzer - report mismatches between assembly files and Go declarations.
//
// • assign.Analyzer - check for useless assignments.
//
// • atomic.Analyzer -  check for common mistakes using the sync/atomic package
//
// • atomicalign.Analyzer -  checks for non-64-bit-aligned arguments to sync/atomic functions
//
// • bools.Analyzer -  check for common mistakes involving boolean operators
//
// • buildtag.Analyzer - check that +build tags are well-formed and correctly located
//
// • cgocall.Analyzer - detect some violations of the cgo pointer passing rules.
//
// • composite.Analyzer - check for unkeyed composite literals
//
// • copylock.Analyzer - check for locks erroneously passed by value
//
// • deepequalerrors.Analyzer - check for the use of reflect.DeepEqual with error values
//
// • errorsas.Analyzer - check that the second argument to errors.As is a pointer to a type implementing error
//
// • httpresponse.Analyzer - check for mistakes using HTTP responses
//
// • loopclosure.Analyzer - check references to loop variables from within nested functions
//
// • lostcancel.Analyzer - check cancel func returned by context.WithCancel is called
//
// • nilfunc.Analyzer - check for useless comparisons between functions and nil
//
// • nilness.Analyzer - inspects the control-flow graph of an SSA function and reports errors such as nil pointer dereferences and degenerate nil pointer comparisons
//
// • printf.Analyzer - check consistency of Printf format strings and arguments
//
// • shadow.Analyzer - check for possible unintended shadowing of variables EXPERIMENTAL
//
// • shift.Analyzer - check for shifts that equal or exceed the width of the integer
//
// • stdmethods.Analyzer - check signature of methods of well-known interfaces
//
// • structtag.Analyzer - check that struct field tags conform to reflect.StructTag.Get
//
// • tests.Analyzer - check for common mistaken usages of tests and examples
//
// • unmarshal.Analyzer - report passing non-pointer or non-interface values to unmarshal
//
// • unreachable.Analyzer - check for unreachable code
//
// • unsafeptr.Analyzer - check for invalid conversions of uintptr to unsafe.Pointer
//
// • unusedresult.Analyzer - check for unused results of calls to some functions
//
// • unusedwrite.Analyzer - check for unused writes
//
// • stringintconv.Analyzer - check for string(int) conversions
//
// • ifaceassert.Analyzer - check for impossible interface-to-interface type assertions
//
// • errcheck.Analyzer - check for unhandled returned errors
//
// • stringlencompare.Analyzer - check string len compare style.
//
// • osexitcheck.OsExitCheckAnalyzer - check for os.Exit() usage in main() functions.
package main

import (
	"strings"

	"github.com/Jay-T/go-devops.git/internal/osexitcheck"
	"github.com/johejo/stringlencompare"
	"github.com/kisielk/errcheck/errcheck"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/multichecker"
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
	"golang.org/x/tools/go/analysis/passes/errorsas"
	"golang.org/x/tools/go/analysis/passes/httpresponse"
	"golang.org/x/tools/go/analysis/passes/ifaceassert"
	"golang.org/x/tools/go/analysis/passes/loopclosure"
	"golang.org/x/tools/go/analysis/passes/lostcancel"
	"golang.org/x/tools/go/analysis/passes/nilfunc"
	"golang.org/x/tools/go/analysis/passes/nilness"
	"golang.org/x/tools/go/analysis/passes/printf"
	"golang.org/x/tools/go/analysis/passes/shadow"
	"golang.org/x/tools/go/analysis/passes/shift"
	"golang.org/x/tools/go/analysis/passes/stdmethods"
	"golang.org/x/tools/go/analysis/passes/stringintconv"
	"golang.org/x/tools/go/analysis/passes/structtag"
	"golang.org/x/tools/go/analysis/passes/tests"
	"golang.org/x/tools/go/analysis/passes/unmarshal"
	"golang.org/x/tools/go/analysis/passes/unreachable"
	"golang.org/x/tools/go/analysis/passes/unsafeptr"
	"golang.org/x/tools/go/analysis/passes/unusedresult"
	"golang.org/x/tools/go/analysis/passes/unusedwrite"
	"honnef.co/go/tools/quickfix"
	"honnef.co/go/tools/simple"
	"honnef.co/go/tools/staticcheck"
	"honnef.co/go/tools/stylecheck"
)

func main() {
	analyzers := []*analysis.Analyzer{
		// report mismatches between assembly files and Go declarations.
		asmdecl.Analyzer,
		// check for useless assignments.
		assign.Analyzer,
		// check for common mistakes using the sync/atomic package
		atomic.Analyzer,
		// checks for non-64-bit-aligned arguments to sync/atomic functions
		atomicalign.Analyzer,
		// check for common mistakes involving boolean operators
		bools.Analyzer,
		// check that +build tags are well-formed and correctly located
		buildtag.Analyzer,
		// detect some violations of the cgo pointer passing rules.
		cgocall.Analyzer,
		// check for unkeyed composite literals
		composite.Analyzer,
		// check for locks erroneously passed by value
		copylock.Analyzer,
		// check for the use of reflect.DeepEqual with error values
		deepequalerrors.Analyzer,
		// check that the second argument to errors.As is a pointer to a type implementing error
		errorsas.Analyzer,
		// check for mistakes using HTTP responses
		httpresponse.Analyzer,
		// check references to loop variables from within nested functions
		loopclosure.Analyzer,
		// check cancel func returned by context.WithCancel is called
		lostcancel.Analyzer,
		// check for useless comparisons between functions and nil
		nilfunc.Analyzer,
		// inspects the control-flow graph of an SSA function and reports errors such as nil pointer dereferences and degenerate nil pointer comparisons
		nilness.Analyzer,
		// check consistency of Printf format strings and arguments
		printf.Analyzer,
		// check for possible unintended shadowing of variables EXPERIMENTAL
		shadow.Analyzer,
		// check for shifts that equal or exceed the width of the integer
		shift.Analyzer,
		// check signature of methods of well-known interfaces
		stdmethods.Analyzer,
		// check that struct field tags conform to reflect.StructTag.Get
		structtag.Analyzer,
		// check for common mistaken usages of tests and examples
		tests.Analyzer,
		// report passing non-pointer or non-interface values to unmarshal
		unmarshal.Analyzer,
		// check for unreachable code
		unreachable.Analyzer,
		// check for invalid conversions of uintptr to unsafe.Pointer
		unsafeptr.Analyzer,
		// check for unused results of calls to some functions
		unusedresult.Analyzer,
		// check for unused writes
		unusedwrite.Analyzer,
		// check for string(int) conversions
		stringintconv.Analyzer,
		// check for impossible interface-to-interface type assertions
		ifaceassert.Analyzer,
		// check for unhandled returned errors
		errcheck.Analyzer,
		// check string len compare style.
		stringlencompare.Analyzer,
		// check for os.Exit() usage in main() functions.
		osexitcheck.OsExitCheckAnalyzer,
	}

	alsoForInclude := map[string]bool{
		// check for not using fmt.Sprintf("%s", x) unnecessarily
		"S1025": true,
		// check for not using Yoda conditions
		"ST1017": true,
		// Converts if/else-if chain to tagged switch
		"QF1003": true,
	}

	for _, v := range staticcheck.Analyzers {
		if strings.HasPrefix(v.Analyzer.Name, "SA") {
			analyzers = append(analyzers, v.Analyzer)
		}
	}

	for _, v := range stylecheck.Analyzers {
		if alsoForInclude[v.Analyzer.Name] {
			analyzers = append(analyzers, v.Analyzer)
		}
	}

	for _, v := range simple.Analyzers {
		if alsoForInclude[v.Analyzer.Name] {
			analyzers = append(analyzers, v.Analyzer)
		}
	}

	for _, v := range quickfix.Analyzers {
		if alsoForInclude[v.Analyzer.Name] {
			analyzers = append(analyzers, v.Analyzer)
		}
	}

	multichecker.Main(analyzers...)
}
