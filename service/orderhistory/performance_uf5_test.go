package orderhistory

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"testing"

	"github.com/beego/beego/v2/client/orm"
	"go_binance_futures/models"
)

func uf5Cycle(t *testing.T, o orm.Ormer, account, owner, symbol, side, asset, hash string, closedAt int64, pnl string) int64 {
	t.Helper()
	p := &models.FuturesManagedPosition{
		AccountID: account, Owner: owner, Symbol: symbol, PositionSide: side,
		Status: "closed", SettlementStatus: "settled", ManagedQty: 0,
		SourceRef: hash, CreatedAt: closedAt - 5000, UpdatedAt: closedAt,
		ClosedAt: closedAt, SettlementProcessedAt: closedAt,
	}
	exact, err := dec(pnl)
	if err != nil {
		t.Fatal(err)
	}
	net := new(big.Rat).Sub(exact, big.NewRat(1, 1))
	p.SettlementPNLSign = net.Sign()
	if _, err = o.Insert(p); err != nil {
		t.Fatal(err)
	}
	for _, leg := range []struct{ intent, qty, pnl, fee string }{
		{"open", "2", "0", "0.5"},
		{"close", "2", pnl, "0.5"},
	} {
		client := fmt.Sprintf("uf5-%d-%s", p.ID, leg.intent)
		orderID := fmt.Sprintf("uf5-ex-%d-%s", p.ID, leg.intent)
		order := &models.FuturesManagedOrder{
			AccountID: account, Owner: owner, Symbol: symbol, PositionSide: side,
			PositionID: &p.ID, ClientOrderID: client, ExchangeOrderID: orderID,
			Intent: leg.intent, Status: "filled", RequestedQty: 2, FilledQty: 2,
			OrderType: "MARKET", CreatedAt: closedAt - 4900, UpdatedAt: closedAt - 200,
		}
		if _, err = o.Insert(order); err != nil {
			t.Fatal(err)
		}
		fill(t, o, account, symbol, side, orderID, fmt.Sprintf("uf5-trade-%d-%s", p.ID, leg.intent),
			client, leg.intent, leg.qty, "100", leg.fee, leg.pnl, asset, p.ID, true)
	}
	if asset != "USDT" {
		if _, err = o.Raw("UPDATE futures_trade_fills SET pnl_asset=? WHERE position_id=?", asset, p.ID).Exec(); err != nil {
			t.Fatal(err)
		}
	}
	return p.ID
}
func uf5Find(t *testing.T, report PerformanceReport, asset string) PerformanceAsset {
	t.Helper()
	for _, row := range report.Assets {
		if row.Asset == asset {
			return row
		}
	}
	t.Fatalf("asset %s missing from %+v", asset, report)
	return PerformanceAsset{}
}
func TestUF5SettledPartialFeesMultipleAssetsSidesRulesAndDrawdown(t *testing.T) {
	o, store := historyDB(t)
	end := store.now().UnixMilli()
	store.ActivatedAt = func(string) (int64, error) { return end - 60000, nil }
	a := strings.Repeat("a", 64)
	b := strings.Repeat("b", 64)
	uf5Cycle(t, o, "main", "auto_strategy", "BTCUSDT", "LONG", "USDT", "auto_strategy:BTCUSDT:"+a, end-8000, "11")
	uf5Cycle(t, o, "main", "auto_strategy", "ETHUSDT", "SHORT", "USDT", "auto_strategy:ETHUSDT:"+a, end-7000, "-4")
	uf5Cycle(t, o, "main", "auto_strategy", "XRPUSDT", "LONG", "USDT", "auto_strategy:XRPUSDT", end-6000, "1")
	uf5Cycle(t, o, "main", "auto_strategy", "SOLUSDC", "SHORT", "USDC", "auto_strategy:SOLUSDC:"+b, end-5000, "3")
	// Two different owners/accounts must never be included.
	uf5Cycle(t, o, "lead", "auto_strategy", "AVAXUSDT", "LONG", "USDT", "auto_strategy:AVAXUSDT:"+a, end-5000, "20")
	uf5Cycle(t, o, "main", "notice_auto_order", "DOGEUSDT", "LONG", "USDT", "other", end-5000, "20")
	report, err := store.Performance(context.Background(), PerformanceFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if report.CandidateCycles != 4 || report.VerifiedCycles != 4 || report.ExcludedCycles != 0 ||
		report.Days != 30 || report.AccountID != "main" || report.Owner != "auto_strategy" || len(report.Assets) != 2 {
		t.Fatalf("account/owner/asset isolation failure: %+v", report)
	}
	usdt := uf5Find(t, report, "USDT")
	if usdt.Trades != 3 || usdt.Wins != 1 || usdt.Losses != 1 || usdt.Zero != 1 ||
		usdt.GrossPnL != "8" || usdt.Commission != "3" || usdt.NetPnL != "5" ||
		usdt.AverageNetPnL != "1.666666666666666667" ||
		usdt.WinRatePct != "33.333333333333333333" || usdt.SettledDrawdown != "5" ||
		usdt.ProfitFactor == nil || *usdt.ProfitFactor != "2" {
		t.Fatalf("USDT realized net/PF/maximum settled net drawdown incorrect: %+v", usdt)
	}
	if len(usdt.ByRule) != 2 || usdt.ByRule[0].Name != a || usdt.ByRule[0].Trades != 2 ||
		usdt.ByRule[1].Name != "unknown" || usdt.ByRule[1].Trades != 1 ||
		len(usdt.BySide) != 2 || len(usdt.BySymbol) != 3 {
		t.Fatalf("rule/side/symbol groups inaccurate: %+v", usdt)
	}
	usdc := uf5Find(t, report, "USDC")
	if usdc.Trades != 1 || usdc.NetPnL != "2" || usdc.ProfitFactor != nil ||
		usdc.ProfitFactorNote != "no_loss_samples" || usdc.SettledDrawdown != "0" {
		t.Fatalf("USDC amount mixed with USDT, or PF fabricated: %+v", usdc)
	}
	filtered, err := store.Performance(context.Background(), PerformanceFilter{Days: 7, RuleHash: a})
	if err != nil || filtered.VerifiedCycles != 2 || len(filtered.Assets) != 1 || filtered.Assets[0].NetPnL != "5" {
		t.Fatalf("same hash across different coins should aggregate together: %+v %v", filtered, err)
	}
	shortOnly, err := store.Performance(context.Background(), PerformanceFilter{PositionSide: "SHORT", Symbol: "SOLUSDC"})
	if err != nil || shortOnly.VerifiedCycles != 1 || shortOnly.Assets[0].Asset != "USDC" {
		t.Fatalf("position side/symbol filtering broken: %+v %v", shortOnly, err)
	}
	unknown, err := store.Performance(context.Background(), PerformanceFilter{RuleHash: "unknown"})
	if err != nil || unknown.VerifiedCycles != 1 || unknown.Assets[0].NetPnL != "0" {
		t.Fatalf("legacy source hash should be unknown, not current template: %+v %v", unknown, err)
	}
}
func TestUF5PartialOpenCloseFillsExactAndManualDisplayOverrideIgnored(t *testing.T) {
	o, store := historyDB(t)
	pid := fullCycle(t, o)
	if _, err := o.Raw("UPDATE futures_managed_positions SET history_status_override='active',history_completeness_override='incomplete' WHERE id=?", pid).Exec(); err != nil {
		t.Fatal(err)
	}
	report, err := store.Performance(context.Background(), PerformanceFilter{Days: 7})
	if err != nil || report.VerifiedCycles != 1 || report.Assets[0].NetPnL != "109.2" ||
		report.Assets[0].Commission != "0.8" || report.Assets[0].GrossPnL != "110" {
		t.Fatalf("full partial fills / display overrides changed certified net: %+v %v", report, err)
	}
	// History presentation intentionally masks financial display, but
	// immutable verified settlement performance must use actual evidence.
	display, err := store.List(context.Background(), Filter{Page: 1, Limit: 20})
	if err != nil || display.List[0].NetPnL != nil {
		t.Fatalf("expected presentation-only override: %+v %v", display, err)
	}
}
func TestUF5ExclusionReasonsDontTurnIncompleteIntoProfit(t *testing.T) {
	cases := []struct {
		name, reason string
		change       func(t *testing.T, o orm.Ormer, id int64)
	}{
		{"legacy", "legacy", func(t *testing.T, o orm.Ormer, id int64) {
			_, err := o.Raw("UPDATE futures_managed_positions SET settlement_status='legacy',settlement_processed_at=0 WHERE id=?", id).Exec()
			if err != nil {
				t.Fatal(err)
			}
		}},
		{"incomplete", "not_settled", func(t *testing.T, o orm.Ormer, id int64) {
			_, err := o.Raw("UPDATE futures_managed_positions SET settlement_status='incomplete',settlement_processed_at=0 WHERE id=?", id).Exec()
			if err != nil {
				t.Fatal(err)
			}
		}},
		{"fee_asset_bnb", "unknown_or_mixed_asset", func(t *testing.T, o orm.Ormer, id int64) {
			_, err := o.Raw("UPDATE futures_trade_fills SET commission_asset='BNB' WHERE position_id=? AND intent='close'", id).Exec()
			if err != nil {
				t.Fatal(err)
			}
		}},
		{"fee_asset_missing", "unknown_or_mixed_asset", func(t *testing.T, o orm.Ormer, id int64) {
			_, err := o.Raw("UPDATE futures_trade_fills SET commission_asset='' WHERE position_id=? AND intent='close'", id).Exec()
			if err != nil {
				t.Fatal(err)
			}
		}},
		{"rest_unverified", "rest_not_verified", func(t *testing.T, o orm.Ormer, id int64) {
			_, err := o.Raw("UPDATE futures_trade_fills SET rest_verified=0 WHERE position_id=? AND intent='close'", id).Exec()
			if err != nil {
				t.Fatal(err)
			}
		}},
		{"missing_fill", "missing_or_mismatched_fills", func(t *testing.T, o orm.Ormer, id int64) {
			_, err := o.Raw("DELETE FROM futures_trade_fills WHERE position_id=? AND intent='close'", id).Exec()
			if err != nil {
				t.Fatal(err)
			}
		}},
		{"wrong_client", "identity_mismatch", func(t *testing.T, o orm.Ormer, id int64) {
			_, err := o.Raw("UPDATE futures_trade_fills SET client_order_id='forged' WHERE position_id=? AND intent='close'", id).Exec()
			if err != nil {
				t.Fatal(err)
			}
		}},
		{"settlement_sign", "incomplete_evidence", func(t *testing.T, o orm.Ormer, id int64) {
			_, err := o.Raw("UPDATE futures_managed_positions SET settlement_pnl_sign=-1 WHERE id=?", id).Exec()
			if err != nil {
				t.Fatal(err)
			}
		}},
		{"managed_nonflat", "settlement_conflict", func(t *testing.T, o orm.Ormer, id int64) {
			_, err := o.Raw("UPDATE futures_managed_positions SET managed_qty=1 WHERE id=?", id).Exec()
			if err != nil {
				t.Fatal(err)
			}
		}},
		{"pre_cutover", "pre_activation", func(t *testing.T, o orm.Ormer, id int64) {
			_, err := o.Raw("UPDATE futures_managed_positions SET created_at=1000 WHERE id=?", id).Exec()
			if err != nil {
				t.Fatal(err)
			}
		}},
		{"closed_before_open", "settlement_conflict", func(t *testing.T, o orm.Ormer, id int64) {
			_, err := o.Raw("UPDATE futures_managed_positions SET created_at=closed_at+1 WHERE id=?", id).Exec()
			if err != nil {
				t.Fatal(err)
			}
		}},
		{"settled_before_close", "settlement_conflict", func(t *testing.T, o orm.Ormer, id int64) {
			_, err := o.Raw("UPDATE futures_managed_positions SET settlement_processed_at=closed_at-1 WHERE id=?", id).Exec()
			if err != nil {
				t.Fatal(err)
			}
		}},
		{"settled_in_future", "settlement_conflict", func(t *testing.T, o orm.Ormer, id int64) {
			_, err := o.Raw("UPDATE futures_managed_positions SET settlement_processed_at=? WHERE id=?", int64(1800000012000), id).Exec()
			if err != nil {
				t.Fatal(err)
			}
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			o, store := historyDB(t)
			id := fullCycle(t, o)
			tc.change(t, o, id)
			report, err := store.Performance(context.Background(), PerformanceFilter{})
			if err != nil || report.CandidateCycles != 1 || report.VerifiedCycles != 0 ||
				report.ExcludedCycles != 1 || report.ExcludedReasons[tc.reason] != 1 || len(report.Assets) != 0 {
				t.Fatalf("unverified cycle leaked PnL or wrong exclusion reason: %+v %v", report, err)
			}
		})
	}
}
func TestUF5ZeroFeeWithoutCurrencyIsAllowedOnlyWhenActuallyZero(t *testing.T) {
	for _, scenario := range []struct{ fee, expected string }{
		{"0", "included"}, {"0.1", "unknown_or_mixed_asset"},
	} {
		t.Run(scenario.fee, func(t *testing.T) {
			o, store := historyDB(t)
			end := store.now().UnixMilli()
			id := uf5Cycle(t, o, "main", "auto_strategy", "BTCUSDT", "LONG", "USDT", "auto_strategy:BTCUSDT", end-4000, "3")
			// Force one leg zero fee but blank fee asset, update sign so that
			// a valid settlement certificate is internally consistent.
			_, err := o.Raw("UPDATE futures_trade_fills SET commission_asset='',commission=? WHERE position_id=? AND intent='open'", scenario.fee, id).Exec()
			if err != nil {
				t.Fatal(err)
			}
			if scenario.fee == "0" {
				_, err = o.Raw("UPDATE futures_managed_positions SET settlement_pnl_sign=1 WHERE id=?", id).Exec()
				if err != nil {
					t.Fatal(err)
				}
			}
			report, err := store.Performance(context.Background(), PerformanceFilter{})
			if err != nil {
				t.Fatal(err)
			}
			if scenario.expected == "included" {
				if report.VerifiedCycles != 1 || report.Assets[0].NetPnL != "2.5" {
					t.Fatalf("zero fee with blank currency shouldn't fail: %+v", report)
				}
			} else if report.VerifiedCycles != 0 || report.ExcludedReasons[scenario.expected] != 1 {
				t.Fatalf("unknown nonzero fee currency must be excluded: %+v", report)
			}
		})
	}
}
func TestUF5DateScopeUsesRealClosedAtAndExactBoundary(t *testing.T) {
	o, store := historyDB(t)
	now := store.now().UTC()
	start7 := now.AddDate(0, 0, -7).UnixMilli()
	start30 := now.AddDate(0, 0, -30).UnixMilli()
	store.ActivatedAt = func(string) (int64, error) { return start30 - 10000, nil }
	uf5Cycle(t, o, "main", "auto_strategy", "BTCUSDT", "LONG", "USDT", "auto_strategy:BTCUSDT", start7, "3")
	uf5Cycle(t, o, "main", "auto_strategy", "ETHUSDT", "LONG", "USDT", "auto_strategy:ETHUSDT", start7-1, "3")
	uf5Cycle(t, o, "main", "auto_strategy", "SOLUSDT", "LONG", "USDT", "auto_strategy:SOLUSDT", start30+10000, "3")
	uf5Cycle(t, o, "main", "auto_strategy", "BNBUSDT", "LONG", "USDT", "auto_strategy:BNBUSDT", now.UnixMilli(), "3")
	// Future-closed impossible candidate is outside the report's end.
	uf5Cycle(t, o, "main", "auto_strategy", "XRPUSDT", "LONG", "USDT", "auto_strategy:XRPUSDT", now.UnixMilli()+1000, "3")
	recent, err := store.Performance(context.Background(), PerformanceFilter{Days: 7})
	if err != nil || recent.VerifiedCycles != 2 {
		t.Fatalf("7d closed_at boundary wrong: %+v %v", recent, err)
	}
	monthly, err := store.Performance(context.Background(), PerformanceFilter{Days: 30})
	if err != nil || monthly.VerifiedCycles != 4 {
		t.Fatalf("30d close scope wrong: %+v %v", monthly, err)
	}
}
func TestUF5FullLossNoSampleAndNeverInventPF(t *testing.T) {
	o, store := historyDB(t)
	empty, err := store.Performance(context.Background(), PerformanceFilter{})
	if err != nil || empty.VerifiedCycles != 0 || len(empty.Assets) != 0 || empty.ExcludedCycles != 0 {
		t.Fatalf("empty sample invented returns: %+v %v", empty, err)
	}
	now := store.now().UnixMilli()
	uf5Cycle(t, o, "main", "auto_strategy", "ETHUSDT", "LONG", "USDT", "auto_strategy:ETHUSDT", now-5000, "-2")
	loss, err := store.Performance(context.Background(), PerformanceFilter{})
	if err != nil || loss.Assets[0].ProfitFactor == nil || *loss.Assets[0].ProfitFactor != "0" ||
		loss.Assets[0].SettledDrawdown != "3" || loss.Assets[0].WinRatePct != "0" {
		t.Fatalf("all-loss PF and drawdown wrong: %+v %v", loss, err)
	}
}
func TestUF5FilterValidationAndHistoryActivationFailClosed(t *testing.T) {
	for _, q := range []PerformanceFilter{
		{Days: 1}, {Days: 31}, {Symbol: "BTC' OR 1=1"}, {Symbol: "BTC%"},
		{PositionSide: "BOTH"}, {RuleHash: "fake"}, {RuleHash: strings.Repeat("x", 64)},
	} {
		_, err := (Store{}).Performance(context.Background(), q)
		if !errors.Is(err, ErrInvalidQuery) {
			t.Fatalf("unsafe filter accepted: %+v %v", q, err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := (Store{}).Performance(ctx, PerformanceFilter{})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled context incorrectly treated as no trades: %v", err)
	}
	o, store := historyDB(t)
	uf5Cycle(t, o, "main", "auto_strategy", "BTCUSDT", "LONG", "USDT", "auto_strategy:BTCUSDT", store.now().UnixMilli()-5000, "3")
	store.ActivatedAt = func(string) (int64, error) { return 0, nil }
	report, err := store.Performance(context.Background(), PerformanceFilter{})
	if err != nil || report.VerifiedCycles != 0 || report.ExcludedReasons["history_not_activated"] != 1 {
		t.Fatalf("unactivated historical ledger counted as verified: %+v %v", report, err)
	}
	store.ActivatedAt = func(string) (int64, error) { return 0, fmt.Errorf("malformed T0") }
	_, err = store.Performance(context.Background(), PerformanceFilter{})
	if err == nil {
		t.Fatal("failed activation parsing must not return an apparently valid empty report")
	}
}
func TestUF5SourceRefStrictRuleExtraction(t *testing.T) {
	a := strings.Repeat("a", 64)
	for _, tc := range []struct{ source, symbol, want string }{
		{"auto_strategy:BTCUSDT:" + a, "BTCUSDT", a},
		{"auto_strategy:BTCUSDT:" + a, "ETHUSDT", "unknown"},
		{"auto_strategy:BTCUSDT:" + strings.Repeat("Z", 64), "BTCUSDT", "unknown"},
		{"auto_strategy:BTCUSDT", "BTCUSDT", "unknown"},
		{"agent_trade:BTCUSDT:" + a, "BTCUSDT", "unknown"},
	} {
		if got := performanceHash(tc.source, tc.symbol); got != tc.want {
			t.Fatalf("source %q got %q, want %q", tc.source, got, tc.want)
		}
	}
}
func TestUF5BatchOver100CyclesNoMissingOrDoubleCount(t *testing.T) {
	o, store := historyDB(t)
	end := store.now().UnixMilli()
	for i := 0; i < 105; i++ {
		uf5Cycle(t, o, "main", "auto_strategy", "BTCUSDT", "LONG", "USDT",
			"auto_strategy:BTCUSDT", end-5000+int64(i), "3")
	}
	report, err := store.Performance(context.Background(), PerformanceFilter{Days: 30})
	if err != nil || report.CandidateCycles != 105 || report.VerifiedCycles != 105 ||
		len(report.Assets) != 1 || report.Assets[0].NetPnL != "210" ||
		report.Assets[0].Trades != 105 || report.Assets[0].WinRatePct != "100" ||
		report.Assets[0].ProfitFactor != nil {
		t.Fatalf("100+ cycle batched read lost/duplicated settled outcomes: %+v, %v", report, err)
	}
}
func TestUF5RejectExactOpenCloseMismatchEvenWithinDisplayTolerance(t *testing.T) {
	o, store := historyDB(t)
	id := uf5Cycle(t, o, "main", "auto_strategy", "BTCUSDT", "LONG",
		"USDT", "auto_strategy:BTCUSDT", store.now().UnixMilli()-5000, "3")
	if _, err := o.Raw("UPDATE futures_trade_fills SET qty='2.0000000001' WHERE position_id=? AND intent='close'", id).Exec(); err != nil {
		t.Fatal(err)
	}
	report, err := store.Performance(context.Background(), PerformanceFilter{Days: 7})
	if err != nil || report.CandidateCycles != 1 || report.VerifiedCycles != 0 ||
		report.ExcludedReasons["missing_or_mismatched_fills"] != 1 {
		t.Fatalf("nonidentical OPEN/CLOSE quantities wrongly certified as flat: %+v %v", report, err)
	}
}

func TestUF5PreservesAcrossSumsAndNoDatabaseMutation(t *testing.T) {
	o, store := historyDB(t)
	id := fullCycle(t, o)
	var before int64
	if err := o.Raw("SELECT COUNT(*) FROM futures_trade_fills WHERE position_id=?", id).QueryRow(&before); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		report, err := store.Performance(context.Background(), PerformanceFilter{})
		if err != nil || report.Assets[0].NetPnL != "109.2" {
			t.Fatalf("changed REST audited ledger: %+v %v", report, err)
		}
	}
	var after int64
	if err := o.Raw("SELECT COUNT(*) FROM futures_trade_fills WHERE position_id=?", id).QueryRow(&after); err != nil {
		t.Fatal(err)
	}
	if before != after {
		t.Fatalf("read-only report mutated trade fills: %d -> %d", before, after)
	}
}
