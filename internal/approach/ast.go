package approach

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"knodexa/internal/model"
	"strconv"
	"strings"
)

func validateASTCheck(check model.ApproachCheck) error {
	if check.MustContain != "" || check.MustNotContain != "" {
		return fmt.Errorf("use separate checks for AST and literal rules")
	}
	if strings.TrimSpace(check.Symbol) == "" {
		return fmt.Errorf("AST check requires a symbol or import path")
	}
	switch check.ASTKind {
	case "require-call", "forbid-call", "forbid-import", "require-function":
		return nil
	default:
		return fmt.Errorf("unsupported AST check kind %s", check.ASTKind)
	}
}
func checkAST(path string, data []byte, check model.ApproachCheck) (bool, string) {
	source, err := parser.ParseFile(token.NewFileSet(), path, data, 0)
	if err != nil {
		return false, "Go source does not parse: " + err.Error()
	}
	imports := map[string]string{}
	found := false
	for _, spec := range source.Imports {
		value, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			return false, err.Error()
		}
		alias := value
		if pos := strings.LastIndex(value, "/"); pos >= 0 {
			alias = value[pos+1:]
		}
		if spec.Name != nil {
			alias = spec.Name.Name
		}
		imports[alias] = value
		if check.ASTKind == "forbid-import" && value == check.Symbol {
			found = true
		}
	}
	ast.Inspect(source, func(node ast.Node) bool {
		switch item := node.(type) {
		case *ast.CallExpr:
			if check.ASTKind != "require-call" && check.ASTKind != "forbid-call" {
				break
			}
			name := ""
			qualified := ""
			switch function := item.Fun.(type) {
			case *ast.Ident:
				name = function.Name
			case *ast.SelectorExpr:
				if receiver, ok := function.X.(*ast.Ident); ok {
					name = receiver.Name + "." + function.Sel.Name
					if imported, ok := imports[receiver.Name]; ok {
						qualified = imported + "." + function.Sel.Name
					}
				}
			}
			if name == check.Symbol || qualified == check.Symbol {
				found = true
			}
		case *ast.FuncDecl:
			if check.ASTKind == "require-function" && item.Name.Name == check.Symbol {
				found = true
			}
		}
		return true
	})
	passed := found
	if check.ASTKind == "forbid-call" || check.ASTKind == "forbid-import" {
		passed = !found
	}
	return passed, fmt.Sprintf("%s %s; AST evidence found=%t", check.ASTKind, check.Symbol, found)
}
