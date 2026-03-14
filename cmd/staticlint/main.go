package main

import (
	"github.com/vsevolod-ryzhov/urlshortner.git/internal/checker/exitchecker"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/multichecker"
	"golang.org/x/tools/go/analysis/passes/copylock"
	"golang.org/x/tools/go/analysis/passes/httpresponse"
	"golang.org/x/tools/go/analysis/passes/printf"
	"golang.org/x/tools/go/analysis/passes/shadow"
	"golang.org/x/tools/go/analysis/passes/structtag"
	"honnef.co/go/tools/quickfix"
	"honnef.co/go/tools/simple"
	"honnef.co/go/tools/staticcheck"
	"honnef.co/go/tools/stylecheck"
)

func main() {
	analyzers := getAnalyzers()
	multichecker.Main(analyzers...)
}

func getAnalyzers() []*analysis.Analyzer {
	analyzers := []*analysis.Analyzer{
		printf.Analyzer,
		shadow.Analyzer,
		structtag.Analyzer,
		httpresponse.Analyzer,
		copylock.Analyzer,
		exitchecker.Analyzer,
	}

	for _, a := range staticcheck.Analyzers {
		analyzers = append(analyzers, a.Analyzer)
	}

	for _, a := range quickfix.Analyzers {
		if a.Analyzer.Name == "QF1007" {
			analyzers = append(analyzers, a.Analyzer)
		}
	}

	for _, a := range stylecheck.Analyzers {
		if a.Analyzer.Name == "ST1023" {
			analyzers = append(analyzers, a.Analyzer)
		}
	}

	for _, a := range simple.Analyzers {
		if a.Analyzer.Name == "S1039" {
			analyzers = append(analyzers, a.Analyzer)
		}
	}

	for _, a := range analyzers {
		if a.Flags.Lookup("buildtags") != nil {
			err := a.Flags.Set("buildtags", "")
			if err != nil {
				panic(err)
			}
		}
	}

	return analyzers
}
