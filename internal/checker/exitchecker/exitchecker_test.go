package exitchecker_test

import (
	"testing"

	"github.com/vsevolod-ryzhov/urlshortner.git/internal/checker/exitchecker"
	"golang.org/x/tools/go/analysis/analysistest"
)

func TestExitChecker(t *testing.T) {
	testdata := analysistest.TestData()

	analysistest.Run(t, testdata, exitchecker.Analyzer, "with_exit")

	analysistest.Run(t, testdata, exitchecker.Analyzer, "without_exit")

	analysistest.Run(t, testdata, exitchecker.Analyzer, "exit_in_function")
}
