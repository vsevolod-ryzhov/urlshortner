package exitchecker_test

import (
	"testing"

	"github.com/vsevolod-ryzhov/urlshortner.git/internal/checker/exitchecker"
	"golang.org/x/tools/go/analysis/analysistest"
)

func TestExitChecker(t *testing.T) {
	testdata := analysistest.TestData()

	analysistest.Run(t, testdata, exitchecker.Analyzer, "exitchecker/exit_in_main")
	analysistest.Run(t, testdata, exitchecker.Analyzer, "exitchecker/exit_not_in_main")
}
