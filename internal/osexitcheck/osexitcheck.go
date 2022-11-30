// The analyzer checks if os.Exit() is not called in main() function.
package osexitcheck

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
)

var OsExitCheckAnalyzer = &analysis.Analyzer{
	Name: "osexitcheck",
	Doc:  "check for os.Exit() usage",
	Run:  run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	checkOsExitExistence := func(x *ast.ExprStmt) {
		c, ok := x.X.(*ast.CallExpr)
		if !ok {
			return
		}

		s, ok := c.Fun.(*ast.SelectorExpr)
		if !ok {
			return
		}

		y, ok := s.X.(*ast.Ident)
		if !ok {
			return
		}

		if y.Name == "os" && s.Sel.Name == "Exit" {
			pass.Reportf(x.Pos(), "should not use os.Exit() in main().")
		}
	}

	checkFuncName := func(x *ast.FuncDecl) bool {
		return x.Name.Name == "main"
	}

	for _, file := range pass.Files {
		isMainFunc := false
		ast.Inspect(file, func(node ast.Node) bool {
			switch x := node.(type) {
			case *ast.FuncDecl:
				isMainFunc = checkFuncName(x)
			case *ast.ExprStmt:
				if isMainFunc {
					checkOsExitExistence(x)
				}
			}
			return true
		})
	}
	return nil, nil
}
