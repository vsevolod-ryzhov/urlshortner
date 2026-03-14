package exitchecker

import (
	"go/ast"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

const doc = `check for direct os.Exit calls in main function

This analyzer reports direct calls to os.Exit in the main function of the main package.`

var Analyzer = &analysis.Analyzer{
	Name:     "exitcheck",
	Doc:      doc,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	if pass.Pkg.Name() != "main" {
		return nil, nil
	}

	var targetFile *ast.File
	for _, f := range pass.Files {
		filename := pass.Fset.File(f.Pos()).Name()

		if isCmdShortenerMainGo(filename) {
			targetFile = f
			break
		}
	}

	if targetFile == nil {
		return nil, nil
	}

	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	var mainFunc *ast.FuncDecl
	inspect.Preorder([]ast.Node{(*ast.FuncDecl)(nil)}, func(n ast.Node) {
		fn := n.(*ast.FuncDecl)

		filename := pass.Fset.File(fn.Pos()).Name()
		if !isCmdShortenerMainGo(filename) {
			return
		}

		if fn.Name.Name == "main" && fn.Recv == nil {
			mainFunc = fn
		}
	})

	if mainFunc == nil {
		return nil, nil
	}

	inspect.Preorder([]ast.Node{(*ast.CallExpr)(nil)}, func(n ast.Node) {
		call := n.(*ast.CallExpr)

		filename := pass.Fset.File(call.Pos()).Name()
		if !isCmdShortenerMainGo(filename) {
			return
		}

		if call.Pos() < mainFunc.Pos() || call.End() > mainFunc.End() {
			return
		}

		if isOSExit(call) {
			pass.Reportf(call.Pos(),
				"direct call to os.Exit in cmd/shortener/cmd_shortener_main.go is not allowed")
		}
	})

	return nil, nil
}

func isCmdShortenerMainGo(filename string) bool {
	if strings.Contains(filename, "cmd_shortener_main.go") { // for testing only
		return true
	}

	absPath, err := filepath.Abs(filename)
	if err != nil {
		return false
	}

	normalizedPath := filepath.ToSlash(absPath)

	if filepath.Base(normalizedPath) != "cmd_shortener_main.go" {
		return false
	}

	if !strings.Contains(normalizedPath, "/cmd/shortener/") {
		return false
	}

	return true
}

func isOSExit(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}

	pkg, ok := sel.X.(*ast.Ident)
	if !ok {
		return false
	}

	return pkg.Name == "os" && sel.Sel.Name == "Exit"
}
