// Package analysis provides a static analyzer for Go code that detects
// calls to panic(), log.Fatal(), and os.Exit() outside the main function.
// This helps enforce safer error handling practices by flagging
// unexpected program exits in library or non-main code.
package analysis

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// Analyzer detects panic() calls and log.Fatal()/os.Exit() outside main.
var Analyzer = &analysis.Analyzer{
	Name:     "exitcheck",
	Doc:      "detects panic() calls and log.Fatal()/os.Exit() outside main",
	Run:      run,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
}

// run performs the analysis.
func run(pass *analysis.Pass) (interface{}, error) {
	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	nodeFilter := []ast.Node{
		(*ast.CallExpr)(nil),
	}

	// Track which functions are main() in main package
	mainFuncs := make(map[*ast.FuncDecl]bool)
	if pass.Pkg.Name() == "main" {
		for _, file := range pass.Files {
			for _, decl := range file.Decls {
				if fd, ok := decl.(*ast.FuncDecl); ok {
					if fd.Name.Name == "main" {
						mainFuncs[fd] = true
					}
				}
			}
		}
	}

	inspect.Preorder(nodeFilter, func(node ast.Node) {
		callExpr := node.(*ast.CallExpr)

		// Check for panic()
		if isPanicCall(callExpr) {
			pass.Reportf(callExpr.Pos(), "panic() detected")
			return
		}

		// Check for log.Fatal() or os.Exit() outside main
		if pass.Pkg.Name() == "main" && !isInMainFunc(callExpr, mainFuncs, pass) {
			if isLogFatalCall(callExpr) {
				pass.Reportf(callExpr.Pos(), "log.Fatal() detected outside main function")
				return
			}
			if isOsExitCall(callExpr) {
				pass.Reportf(callExpr.Pos(), "os.Exit() detected outside main function")
				return
			}
		}
	})

	return nil, nil
}

// isPanicCall checks if the call is to panic().
func isPanicCall(expr *ast.CallExpr) bool {
	if ident, ok := expr.Fun.(*ast.Ident); ok {
		return ident.Name == "panic"
	}
	return false
}

// isLogFatalCall checks if the call is to log.Fatal().
func isLogFatalCall(expr *ast.CallExpr) bool {
	if sel, ok := expr.Fun.(*ast.SelectorExpr); ok {
		if ident, ok := sel.X.(*ast.Ident); ok {
			return ident.Name == "log" && sel.Sel.Name == "Fatal"
		}
	}
	return false
}

// isOsExitCall checks if the call is to os.Exit().
func isOsExitCall(expr *ast.CallExpr) bool {
	if sel, ok := expr.Fun.(*ast.SelectorExpr); ok {
		if ident, ok := sel.X.(*ast.Ident); ok {
			return ident.Name == "os" && sel.Sel.Name == "Exit"
		}
	}
	return false
}

// isInMainFunc checks if a node is inside the main function.
func isInMainFunc(node ast.Node, mainFuncs map[*ast.FuncDecl]bool, pass *analysis.Pass) bool {
	// For each main function, check if node is inside it
	for mainFunc := range mainFuncs {
		if nodeIsInside(mainFunc.Body, node) {
			return true
		}
	}
	return false
}

// nodeIsInside checks if target node is inside the container node.
func nodeIsInside(container ast.Node, target ast.Node) bool {
	if container == target {
		return true
	}

	found := false
	ast.Inspect(container, func(n ast.Node) bool {
		if found {
			return false
		}
		if n == target {
			found = true
			return false
		}
		return true
	})

	return found
}
