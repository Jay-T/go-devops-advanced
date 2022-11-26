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
	expr := func(x *ast.ExprStmt) {
		if c, ok := x.X.(*ast.CallExpr); ok {
			if s, ok := c.Fun.(*ast.SelectorExpr); ok {
				// только функции Println
				if y, ok := s.X.(*ast.Ident); ok {
					if y.Name == "os" && s.Sel.Name == "Exit" {
						pass.Reportf(x.Pos(), "should not use os.Exit() in main().")
					}
				}
			}
		}
	}
	for _, file := range pass.Files {
		ast.Inspect(file, func(node ast.Node) bool {
			switch x := node.(type) {
			case *ast.ExprStmt:
				expr(x)
			}
			return true
		})
	}
	return nil, nil
}
