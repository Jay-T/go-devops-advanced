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
	checkOsExit := func(x *ast.ExprStmt, isMain bool) {
		if c, ok := x.X.(*ast.CallExpr); ok {
			if s, ok := c.Fun.(*ast.SelectorExpr); ok {
				// только функции Println
				if y, ok := s.X.(*ast.Ident); ok {
					if y.Name == "os" && s.Sel.Name == "Exit" && isMain {
						pass.Reportf(x.Pos(), "should not use os.Exit() in main().")
					}
				}
			}
		}
	}

	checkFuncName := func(x *ast.FuncDecl) bool {
		if x.Name.Name == "main" {
			return true
		}
		return false
	}

	for _, file := range pass.Files {
		isMain := false
		ast.Inspect(file, func(node ast.Node) bool {
			switch x := node.(type) {
			case *ast.FuncDecl:
				isMain = checkFuncName(x)
			case *ast.ExprStmt:
				checkOsExit(x, isMain)
			}
			return true
		})
	}
	return nil, nil
}
