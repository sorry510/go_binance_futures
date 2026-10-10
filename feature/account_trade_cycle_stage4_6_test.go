package feature

import (
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/adshao/go-binance/v2/futures"
	binance "go_binance_futures/feature/api/binance"
	"go_binance_futures/feature/strategy"
	"go_binance_futures/models"
	"go_binance_futures/types"
)

func stage46BothDirections(*models.Symbols, []types.FuturesPosition) strategy.OpenResult {
	return strategy.OpenResult{CanLong: true, CanShort: true, LongStrategyHash: "shared-long", ShortStrategyHash: "shared-short"}
}
func TestStage46SharedStrategyConcurrentMainLeadIsolation(t *testing.T) {
	main, mainOpens, _, _ := stage3MockRunner(t, []string{"BTCUSDT"})
	main.AccountID = binance.MainAccountID
	main.Mock = false
	main.LeadSymbols = nil
	main.leadMockPermit = nil
	main.PreflightOpen = nil
	main.Config.FutureMaxCount = 3
	main.SelectCoins = func(_ *models.Config, coins []*models.Symbols) ([]*models.Symbols, error) { return coins[:1], nil }
	main.EvaluateEntry = stage46BothDirections

	lead, leadOpens, _, _ := stage3MockRunner(t, []string{"BTCUSDT"})
	lead.EvaluateEntry = stage46BothDirections
	lead.SelectCoins = func(_ *models.Config, coins []*models.Symbols) ([]*models.Symbols, error) { return coins[:1], nil }
	leadConfigChecks := 0
	lead.PreflightOpen = func(context.Context, *models.Symbols, futures.PositionSideType, float64, float64) error {
		leadConfigChecks++
		return nil
	}
	if err := main.validate(); err != nil {
		t.Fatal(err)
	}
	if err := lead.validate(); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); runAccountTradeCycle(main) }()
	go func() { defer wg.Done(); runAccountTradeCycle(lead) }()
	wg.Wait()
	if *mainOpens != 2 || *leadOpens != 2 || leadConfigChecks != 2 {
		t.Fatalf("shared signals diverged: main=%d lead=%d leadRiskChecks=%d", *mainOpens, *leadOpens, leadConfigChecks)
	}

	// Break Lead only. Main must retain exactly the same two golden directions.
	lead2, leadFaultOpens, _, _ := stage3MockRunner(t, []string{"BTCUSDT"})
	lead2.EvaluateEntry = stage46BothDirections
	lead2.SelectCoins = func(_ *models.Config, coins []*models.Symbols) ([]*models.Symbols, error) { return coins[:1], nil }
	lead2.PreflightOpen = func(context.Context, *models.Symbols, futures.PositionSideType, float64, float64) error {
		return context.DeadlineExceeded
	}
	main2, main2Opens, _, _ := stage3MockRunner(t, []string{"BTCUSDT"})
	main2.AccountID = binance.MainAccountID
	main2.Mock = false
	main2.LeadSymbols = nil
	main2.leadMockPermit = nil
	main2.PreflightOpen = nil
	main2.SelectCoins = func(_ *models.Config, coins []*models.Symbols) ([]*models.Symbols, error) { return coins[:1], nil }
	main2.EvaluateEntry = stage46BothDirections
	wg.Add(2)
	go func() { defer wg.Done(); runAccountTradeCycle(main2) }()
	go func() { defer wg.Done(); runAccountTradeCycle(lead2) }()
	wg.Wait()
	if *leadFaultOpens != 0 || *main2Opens != 2 {
		t.Fatalf("Lead outage contaminated Main: lead=%d main=%d", *leadFaultOpens, *main2Opens)
	}
}
func TestStage46SharedExitSignalsBothSidesEvenWhenLeadPaused(t *testing.T) {
	for _, decision := range []struct {
		name string
		kind tradeExitReason
		want int
	}{
		{"auto", tradeExitAuto, 2},
		{"stop_loss", tradeExitLoss, 2},
		{"take_profit", tradeExitProfit, 2},
		{"hold", tradeExitHold, 0},
	} {
		t.Run(decision.name, func(t *testing.T) {
			r, opens, closes, _ := stage3MockRunner(t, []string{"BTCUSDT"})
			r.AllowNewOpens = false
			r.LeadSymbols.Invalidate()
			r.ReadPositions = func(context.Context) ([]types.FuturesPosition, error) {
				return []types.FuturesPosition{
					{Symbol: "BTCUSDT", Side: "LONG", Amount: "1", Leverage: 4, MarkPrice: "100", UnrealizedProfit: "-1"},
					{Symbol: "BTCUSDT", Side: "SHORT", Amount: "-1", Leverage: 4, MarkPrice: "100", UnrealizedProfit: "1"},
				}, nil
			}
			r.SyncPositions = func(positions []types.FuturesPosition) ([]ownedTradePosition, error) {
				return []ownedTradePosition{
					{Position: positions[0], Owner: "auto_strategy", SourceRef: "stage46_long"},
					{Position: positions[1], Owner: "auto_strategy", SourceRef: "stage46_short"},
				}, nil
			}
			r.EvaluateExit = func(strategy.CloseParams, float64, float64) tradeExitReason { return decision.kind }
			runAccountTradeCycle(r)
			if *closes != decision.want || *opens != 0 {
				t.Fatalf("paused Lead exit signals changed: kind=%s close=%d opens=%d", decision.name, *closes, *opens)
			}
		})
	}
}

// Guard against an accidental production Lead schedule being introduced
// before Stage 5/Stage 7. Scan Go AST, not comments/README, and do not depend
// on remote API availability. This only covers the production entry packages.
func TestStage46NoProductionLeadExecutionEntrypoint(t *testing.T) {
	_, here, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate source file")
	}
	root := filepath.Dir(filepath.Dir(here))
	for _, dir := range []string{"feature", "routers", "controllers"} {
		entries, err := os.ReadDir(filepath.Join(root, dir))
		if err != nil {
			t.Fatal(err)
		}
		for _, entry := range entries {
			name := entry.Name()
			if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
				continue
			}
			path := filepath.Join(root, dir, name)
			file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
			if err != nil {
				t.Fatalf("parse %s: %v", path, err)
			}
			ast.Inspect(file, func(n ast.Node) bool {
				call, yes := n.(*ast.CallExpr)
				if !yes {
					return true
				}
				var ident string
				switch v := call.Fun.(type) {
				case *ast.SelectorExpr:
					ident = v.Sel.Name
				case *ast.Ident:
					ident = v.Name
				}
				switch ident {
				case "NewLeadExecutionAdapter", "ExecuteManagedClose", "ExecuteOpen", "NewReadOnlyExecutor", "leadMockTradeRunner":
					t.Errorf("unexpected production Lead execution invocation in %s: %s", path, ident)
				}
				return true
			})
		}
	}
	// The exported adapter does not provide a hidden alternate production
	// endpoint, and the Stage 3 permit is only implemented in _test.go.
	if _, err := os.Stat(filepath.Join(root, "feature", "account_trade_cycle_stage3_test.go")); err != nil {
		t.Fatalf("test-only Lead permit fixture missing: %v", err)
	}
}

// The initial entrypoint scan is intentionally narrow. This second check
// covers every production Go package including main/command/services, so an
// unintended production-side Lead activation cannot be hidden elsewhere.
func TestStage46RepositoryProductionLeadWritesRemainUnreachable(t *testing.T) {
	_, here, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate source file")
	}
	root := filepath.Dir(filepath.Dir(here))
	seen := 0
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			switch entry.Name() {
			case ".git", "vendor", "node_modules", "static", "dist", "strategy_templates", ".agents", "testdata":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			return nil
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			return err
		}
		seen++
		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			var name string
			switch v := call.Fun.(type) {
			case *ast.SelectorExpr:
				name = v.Sel.Name
			case *ast.Ident:
				name = v.Name
			}
			switch name {
			case "NewLeadExecutionAdapter", "ExecuteManagedClose", "ExecuteOpen", "NewReadOnlyExecutor", "leadMockTradeRunner":
				t.Errorf("Stage 4 production activation detected: %s invokes %s", path, name)
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if seen < 40 {
		t.Fatalf("scan found only %d production Go files", seen)
	}
}

// Gate 4's structural invariant: the production root executable does not
// transitively link the Lead trading package, even via wrappers, interfaces
// or aliases that the AST name scans cannot detect. When Stage 5 intentionally
// integrates read-only Lead state, this test must be reviewed together with
// the new production permission boundary. Never silently delete it.
func TestStage46ProductionBinaryDependencyClosureExcludesLeadAccount(t *testing.T) {
	_, here, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate repository root")
	}
	root := filepath.Dir(filepath.Dir(here))
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "go", "list", "-deps", ".")
	cmd.Dir = root
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("cannot audit production dependency closure: %v (%s)", err, string(output))
	}
	if ctx.Err() != nil {
		t.Fatalf("production dependency audit timed out: %v", ctx.Err())
	}
	lines := strings.Split(string(output), "\n")
	found := 0
	for _, line := range lines {
		pkg := strings.TrimSpace(line)
		if pkg == "go_binance_futures/service/leadaccount" {
			t.Errorf("production binary unexpectedly links Lead execution package: %s", pkg)
		}
		if pkg != "" {
			found++
		}
	}
	if found < 40 {
		t.Fatal(fmt.Sprintf("production dependency closure suspiciously small: %d packages", found))
	}
}
