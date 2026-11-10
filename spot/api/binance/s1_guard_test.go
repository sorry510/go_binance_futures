package binance

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
)

// Every direct Spot SDK Create/Cancel call in the entire spot/ tree must
// have an unconditional top-level guard prior to the exchange operation.
// Unlike a six-function name list, adding a new file/function cannot escape.
func TestS1LegacySpotWritersUsePersistentGate(t *testing.T) {
	fset := token.NewFileSet()
	writers := 0
	err := filepath.WalkDir("../..", func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return err
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			exchangeCreateCalls := 0
			firstExchangePos := token.NoPos
			ast.Inspect(fn.Body, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}
				sel, ok := call.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				if sel.Sel.Name == "NewCreateOrderService" || sel.Sel.Name == "NewCancelOrderService" {
					exchangeCreateCalls++
					if firstExchangePos == token.NoPos || call.Pos() < firstExchangePos {
						firstExchangePos = call.Pos()
					}
				}
				return true
			})
			if exchangeCreateCalls == 0 {
				continue
			}
			writers += exchangeCreateCalls
			guarded := false
			for _, stmt := range fn.Body.List {
				if stmt.Pos() >= firstExchangePos {
					break
				}
				ifstmt, ok := stmt.(*ast.IfStmt)
				if !ok {
					continue
				}
				// Must return on failure in an unconditional top-level IF,
				// not a syntactically present but unreachable nested branch.
				hasReturn := false
				for _, inner := range ifstmt.Body.List {
					if _, ok := inner.(*ast.ReturnStmt); ok {
						hasReturn = true
					}
				}
				if !hasReturn || ifstmt.Init == nil {
					continue
				}
				ast.Inspect(ifstmt.Init, func(n ast.Node) bool {
					call, ok := n.(*ast.CallExpr)
					if !ok {
						return true
					}
					sel, ok := call.Fun.(*ast.SelectorExpr)
					if !ok || (sel.Sel.Name != "AuthorizeLegacyOrder" && sel.Sel.Name != "AuthorizeLiveWrite") || len(call.Args) != 3 {
						return true
					}
					base, ok := sel.X.(*ast.Ident)
					if !ok || base.Name != "spotstrategy" {
						return true
					}
					// Real Spot writers MUST check the actual ctx and
					// persisted order request with an explicit bool intent.
					// Merely mentioning AuthorizeLiveWrite in an unrelated
					// top-level IF no longer satisfies the AST gate.
					if sel.Sel.Name == "AuthorizeLiveWrite" {
						ctx, ok0 := call.Args[0].(*ast.Ident)
						order, ok1 := call.Args[1].(*ast.Ident)
						intent, ok2 := call.Args[2].(*ast.Ident)
						if !ok0 || ctx.Name != "ctx" || !ok1 || order.Name != "req" ||
							!ok2 || (intent.Name != "true" && intent.Name != "false") {
							return true
						}
						if (fn.Name.Name == "Place" && intent.Name == "false") ||
							(fn.Name.Name == "Cancel" && intent.Name == "true") {
							guarded = true
						}
						return true
					}
					// Legacy writers require source + real symbol + explicit intent.
					arg0, ok0 := call.Args[0].(*ast.Ident)
					arg1, ok1 := call.Args[1].(*ast.Ident)
					arg2, ok2 := call.Args[2].(*ast.SelectorExpr)
					if !ok0 || arg0.Name != "source" || !ok1 || arg1.Name != "symbol" || !ok2 {
						return true
					}
					guarded = arg2.Sel.Name == "IntentNewRisk" || arg2.Sel.Name == "IntentExit"
					return true
				})
				if guarded {
					break
				}
			}
			if !guarded {
				t.Errorf("%s.%s: Spot exchange write has no prior unconditional source/symbol/intent gate", path, fn.Name.Name)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if writers != 8 {
		t.Fatalf("expected 8 guarded Spot SDK writers, found %d; review new writers explicitly", writers)
	}
}
