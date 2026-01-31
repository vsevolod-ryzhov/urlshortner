package main

import (
	"go/build"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/multichecker"
	"golang.org/x/tools/go/analysis/passes/printf"
	"golang.org/x/tools/go/analysis/passes/shadow"
	"golang.org/x/tools/go/analysis/passes/structtag"
)

func main() {
	buildCtx := build.Default
	buildCtx.CgoEnabled = false

	analyzers := []*analysis.Analyzer{
		printf.Analyzer,
		shadow.Analyzer,
		structtag.Analyzer,
	}

	for _, a := range analyzers {
		if a.Flags.Lookup("buildtags") != nil {
			a.Flags.Set("buildtags", "")
		}
	}

	multichecker.Main(analyzers...)
}
