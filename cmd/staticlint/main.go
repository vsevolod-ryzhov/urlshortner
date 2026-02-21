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
	"honnef.co/go/tools/staticcheck"
)

func main() {
	analyzers := []*analysis.Analyzer{
		printf.Analyzer,
		shadow.Analyzer,
		structtag.Analyzer,
		httpresponse.Analyzer,
		copylock.Analyzer,
		exitchecker.Analyzer,
	}

	for _, analyzer := range staticcheck.Analyzers {
		if len(analyzer.Analyzer.Name) >= 2 && analyzer.Analyzer.Name[:2] == "SA" {
			analyzers = append(analyzers, analyzer.Analyzer)
		}
		if analyzer.Analyzer.Name == "QF1007" {
			analyzers = append(analyzers, analyzer.Analyzer)
		}
		if analyzer.Analyzer.Name == "ST1023" {
			analyzers = append(analyzers, analyzer.Analyzer)
		}
		if analyzer.Analyzer.Name == "S1039" {
			analyzers = append(analyzers, analyzer.Analyzer)
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

	multichecker.Main(analyzers...)
}
