package feature

import (
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"testing"

	"github.com/adshao/go-binance/v2/futures"
	"go_binance_futures/feature/api/binance"
	"go_binance_futures/feature/strategy"
	"go_binance_futures/models"
	"go_binance_futures/types"
)

// Regression guard for the simulated close loop: a successful close must not
// return from CheckTestResults before later open rows are evaluated.
func TestSimulatedCloseLoopDoesNotReturnAfterOnePosition(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "feature_test_strategy.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name.Name != "CheckTestResults" {
			continue
		}
		var loop *ast.RangeStmt
		ast.Inspect(fn.Body, func(node ast.Node) bool {
			r, ok := node.(*ast.RangeStmt)
			if !ok {
				return true
			}
			x, ok := r.X.(*ast.Ident)
			if ok && x.Name == "results" {
				loop = r
			}
			return true
		})
		if loop == nil {
			t.Fatal("missing simulated positions iteration")
		}
		hasSuccessfulCloseBreak := false
		ast.Inspect(loop.Body, func(node ast.Node) bool {
			if _, ok := node.(*ast.ReturnStmt); ok {
				t.Error("a return within the simulated positions loop skips other eligible closes")
			}
			b, ok := node.(*ast.BlockStmt)
			if !ok || len(b.List) < 2 {
				return true
			}
			for i, stmt := range b.List[:len(b.List)-1] {
				exprStmt, ok := stmt.(*ast.ExprStmt)
				if !ok {
					continue
				}
				call, ok := exprStmt.X.(*ast.CallExpr)
				if !ok {
					continue
				}
				name, ok := call.Fun.(*ast.Ident)
				if !ok || name.Name != "autoTestToTrade" {
					continue
				}
				if next, ok := b.List[i+1].(*ast.BranchStmt); ok && next.Tok == token.BREAK {
					hasSuccessfulCloseBreak = true
				}
			}
			return true
		})
		if !hasSuccessfulCloseBreak {
			t.Fatal("successful strategy close must break only the rule loop")
		}
		return
	}
	t.Fatal("CheckTestResults not found")
}

func TestRealTradeCycleAttemptsAllManagedCloses(t *testing.T) {
	for _, tc := range []struct {
		name      string
		failFirst bool
	}{
		{"all_success", false},
		{"first_close_failed_next_still_attempted", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r, opens, _ := stage3MockRunner(t, []string{"BTCUSDT", "ETHUSDT"})
			r.AccountID = binance.MainAccountID
			r.Mock = false
			r.leadMockPermit = nil
			r.LeadSymbols = nil
			r.AllowNewOpens = false
			positions := []types.FuturesPosition{
				{Symbol: "BTCUSDT", Side: "LONG", Amount: "2", Leverage: 4, MarkPrice: "100", UnrealizedProfit: "-10"},
				{Symbol: "ETHUSDT", Side: "SHORT", Amount: "-3", Leverage: 4, MarkPrice: "100", UnrealizedProfit: "-12"},
			}
			r.ReadPositions = func(context.Context) ([]types.FuturesPosition, error) { return positions, nil }
			r.SyncPositions = func(p []types.FuturesPosition) ([]ownedTradePosition, error) {
				return []ownedTradePosition{
					{Position: p[0], Owner: "auto_strategy", SourceRef: "auto_strategy:BTCUSDT"},
					{Position: p[1], Owner: "auto_strategy", SourceRef: "auto_strategy:ETHUSDT"},
				}, nil
			}
			r.EvaluateEntry = func(*models.Symbols, []types.FuturesPosition) strategy.OpenResult {
				t.Fatal("new opens must not be evaluated while disabled")
				return strategy.OpenResult{}
			}
			r.EvaluateExit = func(strategy.CloseParams, float64, float64) tradeExitReason { return tradeExitAuto }
			var attempted []string
			r.SubmitClose = func(_ string, _ string, symbol string, _ float64, _ futures.PositionSideType) (*futures.CreateOrderResponse, error) {
				attempted = append(attempted, symbol)
				if tc.failFirst && symbol == "BTCUSDT" {
					return nil, fmt.Errorf("simulated first close failure")
				}
				return &futures.CreateOrderResponse{OrderID: int64(len(attempted))}, nil
			}
			runAccountTradeCycle(r)
			if !reflect.DeepEqual(attempted, []string{"BTCUSDT", "ETHUSDT"}) || *opens != 0 {
				t.Fatalf("should attempt both managed closes and no opens: attempts=%v opens=%d", attempted, *opens)
			}
		})
	}
}
