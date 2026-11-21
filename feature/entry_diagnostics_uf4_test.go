package feature

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/adshao/go-binance/v2/futures"

	"go_binance_futures/feature/api/binance"
	"go_binance_futures/feature/strategy"
	"go_binance_futures/models"
	"go_binance_futures/types"
)

func uf4GetReason(symbol, side string) (string, MainEntryDiagnosticsSnapshot) {
	snapshot := MainEntryDiagnostics(symbol, side)
	if len(snapshot.List) == 0 {
		return "", snapshot
	}
	return snapshot.List[0].ReasonCode, snapshot
}

func TestUF4MainEntryDiagnosticActualCycleGates(t *testing.T) {
	cases := []struct {
		name, side, want string
		arrange          func(r *accountTradeRunner)
	}{
		{"no_candidate", "LONG", "", func(r *accountTradeRunner) {
			r.SelectCoins = func(*models.Config, []*models.Symbols) ([]*models.Symbols, error) { return []*models.Symbols{}, nil }
		}},
		{"selector_failure", "LONG", "", func(r *accountTradeRunner) {
			r.SelectCoins = func(*models.Config, []*models.Symbols) ([]*models.Symbols, error) { return nil, errors.New("down") }
		}},
		{"strategy_not_matched", "LONG", "strategy_not_matched", func(r *accountTradeRunner) {
			r.EvaluateEntry = func(*models.Symbols, []types.FuturesPosition) strategy.OpenResult { return strategy.OpenResult{} }
		}},
		{"long_disabled", "LONG", "direction_off", func(r *accountTradeRunner) { r.Config.FutureAllowLong = 0 }},
		{"short_disabled", "SHORT", "direction_off", func(r *accountTradeRunner) { r.Config.FutureAllowShort = 0 }},
		{"max_positions", "LONG", "max_positions", func(r *accountTradeRunner) { r.Config.FutureMaxCount = 0 }},
		{"max_losing_positions", "LONG", "max_losing_positions", func(r *accountTradeRunner) { r.Config.LossMaxCount = 0 }},
		{"existing_position", "LONG", "existing_position", func(r *accountTradeRunner) {
			r.ReadPositions = func(context.Context) ([]types.FuturesPosition, error) {
				return []types.FuturesPosition{{Symbol: "BTCUSDT", Side: "LONG", Amount: "1", UnrealizedProfit: "0"}}, nil
			}
		}},
		{"pending_order", "LONG", "pending_order", func(r *accountTradeRunner) {
			r.ReadOpenOrders = func(context.Context) ([]types.FuturesOrder, error) {
				return []types.FuturesOrder{{Symbol: "BTCUSDT", Side: "BUY", PositionSide: "LONG", Status: "NEW"}}, nil
			}
		}},
		{"depth_unavailable", "LONG", "depth_unavailable", func(r *accountTradeRunner) {
			r.Depth = func(context.Context, string, int) (float64, float64, error) { return 0, 0, errors.New("depth failed") }
		}},
		{"trade_config_failed", "LONG", "trade_config_failed", func(r *accountTradeRunner) {}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r, opens, configs, closes := uf2MainMockOpenRunner(tc.side)
			tc.arrange(r)
			runAccountTradeCycle(r)
			reason, snap := uf4GetReason("BTCUSDT", tc.side)
			if reason != tc.want {
				t.Fatalf("Main cycle reason %q want %q: %+v", reason, tc.want, snap)
			}
			if snap.AccountID != "main" || snap.ObservedAt <= 0 || *opens != 0 || *closes != 0 {
				t.Fatalf("diagnostic modified orders or ownership: %+v open=%d close=%d", snap, *opens, *closes)
			}
			if tc.name == "trade_config_failed" && *configs != 1 {
				t.Fatal("expected actual UF-2 config failure")
			}
			if tc.name == "selector_failure" && snap.GlobalReason != "selector_unavailable" {
				t.Fatalf("selector error wrongly labeled as strategy mismatch: %+v", snap)
			}
			if tc.name == "no_candidate" && (snap.GlobalReason != "no_candidate" || snap.Total != 0) {
				t.Fatalf("no selection must be global, not a fake per-symbol decision: %+v", snap)
			}
		})
	}
}
func TestUF4MainEntrySubmissionIsACKNotFillAndLastTimePreserved(t *testing.T) {
	for _, side := range []string{"LONG", "SHORT"} {
		t.Run(side, func(t *testing.T) {
			r, opens, _, _ := uf2MainMockOpenRunner(side)
			r.EnsureConfig = func(context.Context, *models.Symbols) error { return nil }
			r.SubmitOpen = func(*tradeCycleAccountSnapshot, string, float64, float64, futures.SideType, futures.PositionSideType, futures.OrderType, string) (*futures.CreateOrderResponse, error) {
				*opens++
				return &futures.CreateOrderResponse{OrderID: 42}, nil
			}
			runAccountTradeCycle(r)
			reason, first := uf4GetReason("BTCUSDT", side)
			if reason != "order_submitted" || len(first.List) != 1 || first.List[0].LastOrderSubmittedAt <= 0 || *opens != 1 {
				t.Fatalf("Main ACK does not produce explicit submitted observation: %+v (%d)", first, *opens)
			}
			r.EvaluateEntry = func(*models.Symbols, []types.FuturesPosition) strategy.OpenResult { return strategy.OpenResult{} }
			runAccountTradeCycle(r)
			reason, after := uf4GetReason("BTCUSDT", side)
			if reason != "strategy_not_matched" || after.List[0].LastOrderSubmittedAt != first.List[0].LastOrderSubmittedAt {
				t.Fatalf("latest decision lost previous confirmed order submit time: %+v", after)
			}
		})
	}
}
func TestUF4MainEntryOrderSubmitFailureAndShortConfigFailure(t *testing.T) {
	for _, side := range []string{"LONG", "SHORT"} {
		t.Run(side+"/submit_failed", func(t *testing.T) {
			r, opens, _, _ := uf2MainMockOpenRunner(side)
			r.EnsureConfig = func(context.Context, *models.Symbols) error { return nil }
			r.SubmitOpen = func(*tradeCycleAccountSnapshot, string, float64, float64, futures.SideType, futures.PositionSideType, futures.OrderType, string) (*futures.CreateOrderResponse, error) {
				*opens++
				return nil, errors.New("exchange rejected")
			}
			runAccountTradeCycle(r)
			reason, _ := uf4GetReason("BTCUSDT", side)
			if *opens != 1 || reason != "order_submit_failed" {
				t.Fatalf("%s submission failure incorrectly shown as strategy mismatch or success: %s, calls=%d", side, reason, *opens)
			}
		})
		t.Run(side+"/config_failed", func(t *testing.T) {
			r, opens, _, _ := uf2MainMockOpenRunner(side)
			runAccountTradeCycle(r)
			reason, _ := uf4GetReason("BTCUSDT", side)
			if *opens != 0 || reason != "trade_config_failed" {
				t.Fatalf("%s failed UF-2 gate not captured: %s, calls=%d", side, reason, *opens)
			}
		})
	}
}

// The display is an actual-entry-batch diagnostic, not a listing of all
// configured/disabled symbols. Otherwise hundreds of irrelevant rows crowd
// out the five symbols actually processed by smart_local_v2.
func TestUF4OnlySelectedSymbolsAppearInEntryDiagnostics(t *testing.T) {
	r, _, _, _ := uf2MainMockOpenRunner("LONG")
	var all []*models.Symbols
	for i := 0; i < 320; i++ {
		all = append(all, &models.Symbols{Symbol: fmt.Sprintf("OFF%03dUSDT", i), Enable: 0})
	}
	for i := 0; i < 240; i++ {
		all = append(all, &models.Symbols{Symbol: fmt.Sprintf("SKIP%03dUSDT", i), Enable: 1})
	}
	var selected []*models.Symbols
	for i := 0; i < 5; i++ {
		coin := &models.Symbols{Symbol: fmt.Sprintf("YES%03dUSDT", i), Enable: 1}
		all = append(all, coin)
		selected = append(selected, coin)
	}
	r.LoadSymbols = func() ([]*models.Symbols, error) { return all, nil }
	r.SelectCoins = func(*models.Config, []*models.Symbols) ([]*models.Symbols, error) {
		return selected, nil
	}
	r.EvaluateEntry = func(*models.Symbols, []types.FuturesPosition) strategy.OpenResult {
		return strategy.OpenResult{}
	}
	runAccountTradeCycle(r)
	snap := MainEntryDiagnostics("", "")
	if snap.Total != 10 || snap.GlobalReason != "running" {
		t.Fatalf("only the actual five selected symbols (two directions each) should appear: %+v", snap)
	}
	for _, row := range snap.List {
		if !strings.HasPrefix(row.Symbol, "YES") || row.ReasonCode != "strategy_not_matched" {
			t.Fatalf("nonselected or disabled symbol crowded out real decision: %+v", row)
		}
	}
	for _, sym := range []string{"OFF000USDT", "SKIP000USDT"} {
		if got := MainEntryDiagnostics(sym, ""); got.Total != 0 {
			t.Fatalf("symbol not returned by selector unexpectedly reported: %+v", got)
		}
	}
}

func TestUF4EntryObserverNeverAddsBinanceQueries(t *testing.T) {
	r, _, _, _ := uf2MainMockOpenRunner("LONG")
	reads := 0
	r.ReadPositions = func(context.Context) ([]types.FuturesPosition, error) { reads++; return nil, nil }
	r.ReadOpenOrders = func(context.Context) ([]types.FuturesOrder, error) { reads++; return nil, nil }
	runAccountTradeCycle(r)
	baseline := reads
	for i := 0; i < 100; i++ {
		s := MainEntryDiagnostics("BTCUSDT", "LONG")
		if s.Total != 1 {
			t.Fatal("diagnostics disappeared")
		}
	}
	if reads != baseline {
		t.Fatalf("read-only diagnostic triggered %d additional account calls", reads-baseline)
	}
}
func TestUF4EntryObserverDoesNotOverwriteMainFromLead(t *testing.T) {
	r, _, _, _ := uf2MainMockOpenRunner("LONG")
	runAccountTradeCycle(r)
	before := MainEntryDiagnostics("", "")
	lead := beginMainEntryObservation(&accountTradeRunner{AccountID: binance.LeadAccountID, AllowNewOpens: true})
	lead.record("ETHUSDT", "LONG", "order_submitted")
	lead.global("no_candidate")
	after := MainEntryDiagnostics("", "")
	if before.GlobalReason != after.GlobalReason || before.ObservedAt != after.ObservedAt || before.Total != after.Total {
		t.Fatalf("Lead overwrote Main diagnostics: %+v -> %+v", before, after)
	}
}
func TestUF4MemoryBoundedConcurrentSnapshotAndGeneration(t *testing.T) {
	s := newMainEntryDiagnosticStore()
	start := s.snapshot("", "")
	if start.GlobalReason != "not_started" || start.Total != 0 {
		t.Fatal(start)
	}
	epoch := s.begin(true)
	var wg sync.WaitGroup
	for i := 0; i < 600; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			s.record(epoch, fmt.Sprintf("UF%04dUSDT", i), "LONG", "no_candidate")
			_ = s.snapshot("", "")
		}(i)
	}
	wg.Wait()
	snap := s.snapshot("", "")
	if snap.Total != mainEntryDiagnosticsCap || snap.ObservedAt <= 0 {
		t.Fatalf("memory grew beyond cap: %d", snap.Total)
	}
	for _, v := range snap.List {
		if v.ReasonCode != "no_candidate" || v.ObservedAt <= 0 {
			t.Fatalf("bad timestamp: %+v", v)
		}
	}
	next := s.begin(false)
	s.record(epoch, "STALEUSDT", "LONG", "order_submitted")
	s.record(next, "BTCUSDT", "SHORT", "global_off")
	off := s.snapshot("", "")
	if off.GlobalReason != "global_off" || off.Total != 1 || off.List[0].Symbol != "BTCUSDT" {
		t.Fatalf("late writer from prior cycle contaminated latest snapshot: %+v", off)
	}
	s.record(next, "BTCUSDT", "SHORT", "order_submitted")
	submitted := s.snapshot("BTCUSDT", "SHORT")
	if submitted.Total != 1 || submitted.List[0].LastOrderSubmittedAt <= 0 {
		t.Fatal(submitted)
	}
	s.begin(true)
	s.record(s.generation, "BTCUSDT", "SHORT", "strategy_not_matched")
	last := s.snapshot("BTCUSDT", "SHORT")
	if last.Total != 1 || last.List[0].LastOrderSubmittedAt != submitted.List[0].LastOrderSubmittedAt {
		t.Fatal(last)
	}
	if len(s.lastSubmitted) > mainEntryDiagnosticsCap {
		t.Fatalf("unbounded submission history: %d", len(s.lastSubmitted))
	}
}
func TestUF4ExplicitUnavailabilityIsNotStrategyNotMatched(t *testing.T) {
	r, _, _, _ := uf2MainMockOpenRunner("LONG")
	r.ReadPositions = func(context.Context) ([]types.FuturesPosition, error) { return nil, errors.New("account unavailable") }
	r.Sleep = func(time.Duration) {}
	runAccountTradeCycle(r)
	s := MainEntryDiagnostics("BTCUSDT", "LONG")
	if s.GlobalReason != "positions_unavailable" || s.Total != 0 {
		t.Fatalf("no candidate batch was selected before account failure: %+v", s)
	}
	r.LoadSymbols = func() ([]*models.Symbols, error) { return nil, errors.New("local missing") }
	runAccountTradeCycle(r)
	s = MainEntryDiagnostics("", "")
	if s.GlobalReason != "symbols_unavailable" || s.Total != 0 {
		t.Fatal(s)
	}
}
func TestUF4DoesNotInferVolumeFromNoCandidate(t *testing.T) {
	r, _, _, _ := uf2MainMockOpenRunner("LONG")
	r.SelectCoins = func(*models.Config, []*models.Symbols) ([]*models.Symbols, error) { return []*models.Symbols{}, nil }
	runAccountTradeCycle(r)
	s := MainEntryDiagnostics("BTCUSDT", "")
	if s.Total != 0 || s.GlobalReason != "no_candidate" {
		t.Fatalf("absent selection must not invent a per-symbol filter reason: %+v", s)
	}
}
