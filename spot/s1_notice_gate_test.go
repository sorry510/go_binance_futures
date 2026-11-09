package spot

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

// A blocked legacy auto SELL may not consume its has_notice/enable flags.
// Keep this as an order-sensitive test, not a string-contains assertion.
func TestS1NoticeOrderGatePrecedesSignalConsumption(t *testing.T) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "spot.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	var firstGate, firstConsumed token.Pos
	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil || fn.Name.Name != "NoticeAndAutoOrder" {
			continue
		}
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			switch node := n.(type) {
			case *ast.CallExpr:
				sel, ok := node.Fun.(*ast.SelectorExpr)
				if ok && sel.Sel.Name == "AuthorizeLegacyOrder" {
					if firstGate == token.NoPos || node.Pos() < firstGate {
						firstGate = node.Pos()
					}
				}
			case *ast.AssignStmt:
				for _, lhs := range node.Lhs {
					sel, ok := lhs.(*ast.SelectorExpr)
					if !ok || sel.Sel.Name != "HasNotice" {
						continue
					}
					if base, ok := sel.X.(*ast.Ident); ok && base.Name == "coin" {
						if firstConsumed == token.NoPos || node.Pos() < firstConsumed {
							firstConsumed = node.Pos()
						}
					}
				}
			}
			return true
		})
	}
	if firstGate == token.NoPos || firstConsumed == token.NoPos {
		t.Fatalf("notice gate or has_notice mutation missing gate=%d consumed=%d", firstGate, firstConsumed)
	}
	if firstGate >= firstConsumed {
		t.Fatalf("legacy Spot auto order signal consumed before mode gate at %s / %s", fset.Position(firstConsumed), fset.Position(firstGate))
	}
}
