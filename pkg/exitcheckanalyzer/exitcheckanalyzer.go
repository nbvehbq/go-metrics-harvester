// package exitcheckanalyzer defines an Analyzer that that checks
// for the use os.Exit in main function
package exitcheckanalyzer

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
)

var ExitCheckAnalyzer = &analysis.Analyzer{
	Name: "exitheck",
	Doc:  "check for os.Exit call in main function",
	Run:  run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	funcDecl := func(x *ast.FuncDecl) {
		if x.Name.Name == "main" {
			for _, stmt := range x.Body.List {
				call, ok := stmt.(*ast.ExprStmt)
				if !ok {
					continue
				}
				callexp, ok := call.X.(*ast.CallExpr)
				if !ok {
					continue
				}
				sel, ok := callexp.Fun.(*ast.SelectorExpr)
				if !ok {
					continue
				}
				if sel.Sel.Name == "Exit" {
					if checkRunInArgument(callexp) {
						pass.Reportf(call.Pos(), "os.Exit call in main function")
					}
				}
			}
		}
	}
	for _, file := range pass.Files {
		ast.Inspect(file, func(node ast.Node) bool {
			switch x := node.(type) {
			case *ast.FuncDecl:
				funcDecl(x)
			}
			return true
		})
	}
	return nil, nil
}

func checkRunInArgument(exp *ast.CallExpr) bool {
	for _, arg := range exp.Args {
		call, ok := arg.(*ast.CallExpr)
		if !ok {
			continue
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			continue
		}
		// ident, ok := exp.X.(*ast.Ident)
		// if !ok {
		// 	continue
		// }
		if sel.Sel.Name == "Run" {
			return false
		}
		// ast.Print(nil, ident)
	}
	return true
}
