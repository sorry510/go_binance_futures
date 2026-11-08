package futuresownership

import (
	"context"
	"errors"
	"fmt"
	"github.com/adshao/go-binance/v2/futures"
	"strings"
	"testing"

	"github.com/beego/beego/v2/client/orm"
	"go_binance_futures/models"
	"go_binance_futures/service/tradefills"
)

func settlementConfig(t *testing.T, autoScale, autoTest int) {
	t.Helper()
	o := orm.NewOrm()
	if _, err := o.Raw("DELETE FROM config").Exec(); err != nil {
		t.Fatal(err)
	}
	cfg := models.Config{ID: 1, Version: 22, LossMaxCount: 5, LossAutoScale: autoScale, FutureTestAutoTradeCountLimit: autoTest,
		FutureEnable: 1, FutureAllowLong: 1, FutureAllowShort: 1}
	if _, err := o.Insert(&cfg); err != nil {
		t.Fatal(err)
	}
}

func settlementFilled(t *testing.T, s Service, intent, clientID, orderID string, qty float64, price float64, pnl, fee string) models.FuturesManagedPosition {
	t.Helper()
	ctx := context.Background()
	if intent == IntentOpen {
		claimOpen(t, s, OwnerAutoStrategy, "BTCUSDT", "LONG", clientID, qty)
	} else {
		_, err := s.ClaimOrder(ctx, ClaimOrderInput{Owner: OwnerAutoStrategy, Symbol: "BTCUSDT", PositionSide: "LONG",
			Intent: intent, ClientOrderID: clientID, RequestedQty: qty, OrderType: "MARKET"})
		if err != nil {
			t.Fatal(err)
		}
	}
	if err := s.MarkOrderSubmitted(ctx, clientID, orderID); err != nil {
		t.Fatal(err)
	}
	pos, err := s.ApplyFill(ctx, clientID, qty, price)
	if err != nil {
		t.Fatal(err)
	}
	tid := strings.TrimPrefix(orderID, "1")
	if tid == "" {
		tid = orderID
	}
	fill := models.FuturesTradeFill{Market: tradefills.MarketUSDM, AccountID: "main", Symbol: "BTCUSDT",
		OrderID: orderID, TradeID: tid, PositionSide: "LONG", Qty: fmt.Sprintf("%.8f", qty),
		Price: fmt.Sprintf("%.8f", price), Commission: fee, CommissionAsset: "USDT", RealizedPnL: pnl,
		TradeTime: 1800000000000, RESTVerified: true}
	if err = tradefills.Insert(orm.NewOrm(), fill); err != nil {
		t.Fatal(err)
	}
	return pos
}
func settlementReconciler(s Service, margin string) Reconciler {
	return Reconciler{
		Ownership: s, Executor: Executor{Ownership: s, Broker: &fakeOrderBroker{}},
		Account:               fakeAccountPositionSource{positions: nil},
		SettlementMarginAsset: func(_ context.Context, _ string) (string, error) { return margin, nil },
	}
}
func loadSettlementPosition(t *testing.T, id int64) models.FuturesManagedPosition {
	t.Helper()
	var row models.FuturesManagedPosition
	if err := orm.NewOrm().QueryTable(new(models.FuturesManagedPosition)).Filter("id", id).One(&row); err != nil {
		t.Fatal(err)
	}
	return row
}
func loadSettlementConfig(t *testing.T) models.Config {
	t.Helper()
	var row models.Config
	if err := orm.NewOrm().QueryTable(new(models.Config)).Filter("id", 1).One(&row); err != nil {
		t.Fatal(err)
	}
	return row
}

func TestStage013PartialClosesOnlyOneVerifiedSettlement(t *testing.T) {
	prepareOwnershipDB(t)
	settlementConfig(t, 1, 2)
	s := testService()
	ctx := context.Background()
	p := settlementFilled(t, s, IntentOpen, "settle_open", "1101", 4, 100, "0", "0.20")
	if p.SettlementStatus != SettlementOpen {
		t.Fatalf("new cycle not marked open: %+v", p)
	}
	part := settlementFilled(t, s, IntentClose, "settle_partial", "1201", 1, 110, "10", "0.05")
	if part.ManagedQty != 3 || part.Status != PositionActive {
		t.Fatalf("partially closed early: %+v", part)
	}
	// Even if caller claims the exchange is flat, an active cycle cannot settle.
	result, err := s.settleVerifiedFlat(ctx, p.ID, "USDT")
	if err != nil || result.Status == SettlementSettled {
		t.Fatalf("active cycle settled: %+v %v", result, err)
	}
	p = settlementFilled(t, s, IntentClose, "settle_final", "1202", 3, 120, "60", "0.15")
	if p.Status != PositionClosed || p.SettlementStatus != SettlementIncomplete {
		t.Fatalf("not incomplete at close: %+v", p)
	}
	before := loadSettlementConfig(t)
	if before.SettlementScaleBalance != 0 {
		t.Fatal("risk triggered before verified flat")
	}
	r := settlementReconciler(s, "USDT")
	summary, err := r.ReconcileOwner(ctx, OwnerAutoStrategy)
	if err != nil || summary.OrdersUnresolved != 0 {
		t.Fatalf("verified reconcile failed: %+v %v", summary, err)
	}
	settled := loadSettlementPosition(t, p.ID)
	if settled.SettlementStatus != SettlementSettled || settled.SettlementProcessedAt == 0 || settled.SettlementPNLSign != 1 {
		t.Fatalf("valid settlement not persisted: %+v", settled)
	}
	cfg := loadSettlementConfig(t)
	if cfg.SettlementScaleBalance != 1 || cfg.SettlementConsecutiveLosses != 0 || cfg.LossMaxCount != 5 {
		t.Fatalf("risk should update once: %+v", cfg)
	}
	// Reboot/repeat simulation: many reconciles and direct settlement calls
	// cannot re-apply the old cycle even with new service instances.
	for i := 0; i < 20; i++ {
		again, err := s.settleVerifiedFlat(ctx, p.ID, "USDT")
		if err != nil || again.Status != SettlementSettled {
			t.Fatalf("idempotent settlement #%d: %+v %v", i, again, err)
		}
	}
	cfg = loadSettlementConfig(t)
	if cfg.SettlementScaleBalance != 1 || cfg.LossMaxCount != 5 {
		t.Fatalf("risk double-applied: %+v", cfg)
	}
	var asset string
	if err := orm.NewOrm().Raw("SELECT pnl_asset FROM futures_trade_fills WHERE order_id='1202'").QueryRow(&asset); err != nil || asset != "USDT" {
		t.Fatalf("verified contract settlement asset not stored: %s %v", asset, err)
	}
}

func TestStage013IncompleteExternalReductionAndAssetMismatch(t *testing.T) {
	for _, tc := range []string{"missing_close", "bnb_fee", "invalid_asset"} {
		t.Run(tc, func(t *testing.T) {
			prepareOwnershipDB(t)
			settlementConfig(t, 1, 2)
			s := testService()
			pos := settlementFilled(t, s, IntentOpen, "settle_gap_open", "1301", 3, 100, "0", "0.15")
			if tc == "missing_close" {
				// Account observed zero twice, but no owned close Fill exists.
				observed, err := s.ReconcilePosition(context.Background(), OwnerAutoStrategy, "BTCUSDT", "LONG", 0)
				if err != nil || observed.Status != PositionClosed {
					t.Fatalf("external zero: %+v %v", observed, err)
				}
			} else {
				pos = settlementFilled(t, s, IntentClose, "settle_gap_close", "1302", 3, 105, "15", "0.10")
				if tc == "bnb_fee" {
					if _, err := orm.NewOrm().Raw("UPDATE futures_trade_fills SET commission_asset='BNB' WHERE order_id='1302'").Exec(); err != nil {
						t.Fatal(err)
					}
				}
			}
			r := settlementReconciler(s, "USDT")
			if tc == "invalid_asset" {
				r.SettlementMarginAsset = func(context.Context, string) (string, error) { return "UNKNOWN", nil }
			}
			_, err := r.ReconcileOwner(context.Background(), OwnerAutoStrategy)
			if err != nil {
				t.Fatal(err)
			}
			after := loadSettlementPosition(t, pos.ID)
			if after.SettlementStatus != SettlementIncomplete || after.SettlementProcessedAt != 0 {
				t.Fatalf("unproven cycle settled: %+v", after)
			}
			cfg := loadSettlementConfig(t)
			if cfg.SettlementScaleBalance != 0 || cfg.SettlementConsecutiveLosses != 0 {
				t.Fatalf("incomplete cycle triggered risk: %+v", cfg)
			}
		})
	}
}

func TestStage013DurableLossScaleAndAutoTestAfterTwoLosses(t *testing.T) {
	prepareOwnershipDB(t)
	settlementConfig(t, 1, 2)
	s := testService()
	ctx := context.Background()
	for cycle := 0; cycle < 2; cycle++ {
		p := settlementFilled(t, s, IntentOpen, fmt.Sprintf("loss_open_%d", cycle), fmt.Sprintf("1%d01", cycle+4), 1, 100, "0", "0.10")
		p = settlementFilled(t, s, IntentClose, fmt.Sprintf("loss_close_%d", cycle), fmt.Sprintf("1%d02", cycle+4), 1, 90, "-10", "0.10")
		if p.Status != PositionClosed {
			t.Fatal("expected cycle close")
		}
		r := settlementReconciler(s, "USDT")
		if _, err := r.ReconcileOwner(ctx, OwnerAutoStrategy); err != nil {
			t.Fatal(err)
		}
		cfg := loadSettlementConfig(t)
		if cycle == 0 {
			if cfg.SettlementScaleBalance != -1 || cfg.SettlementConsecutiveLosses != 1 || cfg.LossMaxCount != 5 || cfg.FutureAllowLong != 1 {
				t.Fatalf("first loss disabled early: %+v", cfg)
			}
		}
	}
	cfg := loadSettlementConfig(t)
	if cfg.SettlementScaleBalance != 0 || cfg.SettlementConsecutiveLosses != 0 || cfg.LossMaxCount != 4 ||
		cfg.FutureEnable != 1 || cfg.FutureAllowLong != 0 || cfg.FutureAllowShort != 0 || cfg.FutureTest != 1 {
		t.Fatalf("two actual net losses must adjust risk precisely once: %+v", cfg)
	}
}

func TestStage013SettlementDBFailureRollsBackRiskAndMarker(t *testing.T) {
	prepareOwnershipDB(t)
	settlementConfig(t, 1, 2)
	s := testService()
	ctx := context.Background()
	pos := settlementFilled(t, s, IntentOpen, "rollback_open", "1701", 1, 100, "0", "0.05")
	pos = settlementFilled(t, s, IntentClose, "rollback_close", "1702", 1, 110, "10", "0.05")
	o := orm.NewOrm()
	if _, err := o.Raw(`CREATE TRIGGER fail_s510_settlement BEFORE UPDATE ON config
      BEGIN SELECT RAISE(ABORT,'forced config failure'); END`).Exec(); err != nil {
		t.Fatal(err)
	}
	result, err := s.settleVerifiedFlat(ctx, pos.ID, "USDT")
	if _, dropErr := o.Raw("DROP TRIGGER IF EXISTS fail_s510_settlement").Exec(); dropErr != nil {
		t.Fatal(dropErr)
	}
	if err == nil {
		t.Fatalf("expected atomic risk rollback: %+v", result)
	}
	after := loadSettlementPosition(t, pos.ID)
	if after.SettlementProcessedAt != 0 || after.SettlementStatus != SettlementIncomplete {
		t.Fatalf("config failure left settled marker: %+v", after)
	}
	cfg := loadSettlementConfig(t)
	if cfg.SettlementScaleBalance != 0 || cfg.LossMaxCount != 5 {
		t.Fatalf("partial risk config mutation: %+v", cfg)
	}
	if result, err = s.settleVerifiedFlat(ctx, pos.ID, "USDT"); err != nil || result.Status != SettlementSettled {
		t.Fatalf("recovery after rollback failed: %+v %v", result, err)
	}
}

func TestStage013OldClosedCycleCannotSettleOrMutateConfig(t *testing.T) {
	prepareOwnershipDB(t)
	settlementConfig(t, 1, 2)
	s := testService()
	// Old production Managed rows acquire default legacy; never backfill.
	old := models.FuturesManagedPosition{AccountID: "main", Owner: OwnerAutoStrategy,
		Symbol: "BTCUSDT", PositionSide: "LONG", Status: PositionClosed, ManagedQty: 0, CreatedAt: 1, ClosedAt: 20, UpdatedAt: 20}
	id, err := orm.NewOrm().Insert(&old)
	if err != nil {
		t.Fatal(err)
	}
	result, err := s.settleVerifiedFlat(context.Background(), id, "USDT")
	if err != nil || result.Status == SettlementSettled {
		t.Fatalf("old cycle must be excluded: %+v %v", result, err)
	}
	cfg := loadSettlementConfig(t)
	if cfg.SettlementScaleBalance != 0 || cfg.LossMaxCount != 5 {
		t.Fatalf("old history triggered risk: %+v", cfg)
	}
}

func TestStage013MissingContractMetadataCannotSettle(t *testing.T) {
	prepareOwnershipDB(t)
	settlementConfig(t, 1, 2)
	s := testService()
	pos := settlementFilled(t, s, IntentOpen, "missing_asset_open", "1801", 1, 100, "0", "0.1")
	pos = settlementFilled(t, s, IntentClose, "missing_asset_close", "1802", 1, 110, "10", "0.1")
	r := settlementReconciler(s, "")
	r.SettlementMarginAsset = func(context.Context, string) (string, error) {
		return "", errors.New("Binance exchangeInfo unavailable")
	}
	summary, err := r.ReconcileOwner(context.Background(), OwnerAutoStrategy)
	if err != nil || summary.OrdersUnresolved == 0 {
		t.Fatalf("missing asset verification should defer: %+v %v", summary, err)
	}
	after := loadSettlementPosition(t, pos.ID)
	if after.SettlementStatus != SettlementIncomplete || after.SettlementProcessedAt != 0 {
		t.Fatalf("missing contract metadata settled: %+v", after)
	}
}

func TestStage013ManualClaimWithoutFillsRemainsIncomplete(t *testing.T) {
	prepareOwnershipDB(t)
	settlementConfig(t, 1, 2)
	s := testService()
	// Manual ownership has no provenance of the original OPEN fills;
	// periodic zero cannot invent trades or debit risk streaks.
	r := manualClaimReconciler(manualHype())
	claim := manualClaimFixture()
	claim.Owner = OwnerAutoStrategy
	pos, err := r.ManualClaimPosition(context.Background(), claim)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.ReconcilePosition(context.Background(), OwnerAutoStrategy, pos.Symbol, pos.PositionSide, 0); err != nil {
		t.Fatal(err)
	}
	result, err := s.settleVerifiedFlat(context.Background(), pos.ID, "USDT")
	if err != nil || result.Status == SettlementSettled {
		t.Fatalf("manual claim history fabricated: %+v %v", result, err)
	}
}

// A positive gross trade does not equal a profitable cycle after real costs.
func TestStage013PositiveGrossButNegativeNetCountsAsLoss(t *testing.T) {
	prepareOwnershipDB(t)
	settlementConfig(t, 1, 2)
	s := testService()
	p := settlementFilled(t, s, IntentOpen, "net_open", "1901", 1, 100, "0", "0.60")
	p = settlementFilled(t, s, IntentClose, "net_close", "1902", 1, 101, "1", "0.60")
	if p.Status != PositionClosed {
		t.Fatal("cycle must close")
	}
	r := settlementReconciler(s, "USDT")
	if _, err := r.ReconcileOwner(context.Background(), OwnerAutoStrategy); err != nil {
		t.Fatal(err)
	}
	pos := loadSettlementPosition(t, p.ID)
	cfg := loadSettlementConfig(t)
	if pos.SettlementPNLSign != -1 || cfg.SettlementConsecutiveLosses != 1 || cfg.SettlementScaleBalance != -1 {
		t.Fatalf("gross +1, fees 1.20, net -0.20 should count as one loss: pos=%+v cfg=%+v", pos, cfg)
	}
}

// R2-N2: a FILLED order with unverified WS money facts is not in
// ActiveOrders. Its closed cycle must retry REST via the pinned order ID,
// without re-submitting or reopening a previous trading cycle.
func TestStageR2ClosedFilledOrderProvisionalRESTAutoRecovers(t *testing.T) {
	prepareOwnershipDB(t)
	settlementConfig(t, 1, 2)
	s := testService()
	ctx := context.Background()
	p := settlementFilled(t, s, IntentOpen, "r2_rest_open", "1101", 1, 100, "0", "0.05")
	p = settlementFilled(t, s, IntentClose, "r2_rest_close", "1201", 1, 110, "10", "0.07")
	if p.Status != PositionClosed {
		t.Fatal("expected closed cycle")
	}
	active, err := s.ActiveOrders(ctx, OwnerAutoStrategy)
	if err != nil || len(active) != 0 {
		t.Fatalf("fixture must have no active order: %+v %v", active, err)
	}
	o := orm.NewOrm()
	if _, err := o.Raw("UPDATE futures_trade_fills SET rest_verified=0,commission='0',commission_asset='' WHERE order_id='1201'").Exec(); err != nil {
		t.Fatal(err)
	}
	broker := &stage012Broker{tradeError: errors.New("REST unavailable")}
	r := settlementReconciler(s, "USDT")
	r.Executor.Broker = broker
	first, err := r.ReconcileOwner(ctx, OwnerAutoStrategy)
	if err != nil || first.OrdersUnresolved == 0 {
		t.Fatalf("missing REST confirmation must defer settlement: %+v %v", first, err)
	}
	if got := loadSettlementPosition(t, p.ID); got.SettlementStatus != SettlementIncomplete {
		t.Fatalf("unverified cycle marked settled: %+v", got)
	}
	broker.tradeError = nil
	broker.trades = []*futures.AccountTrade{ledgerTrade(201, 1201, "1", "110", "0.07", "10", futures.PositionSideTypeLong)}
	second, err := r.ReconcileOwner(ctx, OwnerAutoStrategy)
	if err != nil || second.OrdersUnresolved != 0 {
		t.Fatalf("inactive order not backfilled: %+v %v", second, err)
	}
	got := loadSettlementPosition(t, p.ID)
	if got.SettlementStatus != SettlementSettled || got.SettlementPNLSign != 1 {
		t.Fatalf("incomplete not recovered: %+v", got)
	}
	var verified int
	if err := o.Raw("SELECT rest_verified FROM futures_trade_fills WHERE order_id='1201'").QueryRow(&verified); err != nil || verified != 1 {
		t.Fatalf("official REST confirmation not persisted: %d %v", verified, err)
	}
	// A second reconciliation after settlement cannot replay risk.
	before := loadSettlementConfig(t)
	if _, err = r.ReconcileOwner(ctx, OwnerAutoStrategy); err != nil {
		t.Fatal(err)
	}
	after := loadSettlementConfig(t)
	if before.SettlementScaleBalance != after.SettlementScaleBalance || before.LossMaxCount != after.LossMaxCount {
		t.Fatalf("backfill settlement replayed risk: before=%+v after=%+v", before, after)
	}
}

func TestStageR2VerifiedZeroFeeWithoutAssetSettlesCorrectly(t *testing.T) {
	prepareOwnershipDB(t)
	settlementConfig(t, 1, 2)
	s := testService()
	ctx := context.Background()
	p := settlementFilled(t, s, IntentOpen, "r2_zero_fee_open", "1101", 1, 100, "0", "0")
	p = settlementFilled(t, s, IntentClose, "r2_zero_fee_close", "1201", 1, 110, "10", "0.1")
	if _, err := orm.NewOrm().Raw("UPDATE futures_trade_fills SET commission_asset='' WHERE order_id='1101'").Exec(); err != nil {
		t.Fatal(err)
	}
	r := settlementReconciler(s, "USDT")
	summary, err := r.ReconcileOwner(ctx, OwnerAutoStrategy)
	if err != nil || summary.OrdersUnresolved != 0 {
		t.Fatalf("REST-confirmed zero fee without asset failed: %+v %v", summary, err)
	}
	settled := loadSettlementPosition(t, p.ID)
	if settled.SettlementStatus != SettlementSettled || settled.SettlementPNLSign != 1 {
		t.Fatalf("verified zero-fee cycle not settled: %+v", settled)
	}
}

// R3-N1 production-shaped source: missing exact ZERO must NOT starve the
// independent read-only REST Fill backfill, but must still block settlement.
func TestStageR3MissingExactZeroBackfillsFillsWithoutSettling(t *testing.T) {
	prepareOwnershipDB(t)
	settlementConfig(t, 1, 2)
	svc := testService()
	opened := settlementFilled(t, svc, IntentOpen, "r3_open", "1101", 1, 100, "0", "0.02")
	closed := settlementFilled(t, svc, IntentClose, "r3_close", "1201", 1, 110, "10", "0.03")
	if opened.ID != closed.ID || closed.SettlementStatus != SettlementIncomplete {
		t.Fatalf("invalid closed test cycle: %+v", closed)
	}
	o := orm.NewOrm()
	if _, err := o.Raw("UPDATE futures_trade_fills SET rest_verified=0, commission='0', commission_asset='' WHERE order_id='1201'").Exec(); err != nil {
		t.Fatal(err)
	}
	broker := &stage012Broker{trades: []*futures.AccountTrade{ledgerTrade(201, 1201, "1", "110", "0.03", "10", futures.PositionSideTypeLong)}}
	source := reviewPartialAccountSnapshot{fail: map[string]bool{"BTCUSDT|LONG": true}}
	r := Reconciler{Ownership: svc, Executor: Executor{Ownership: svc, Broker: broker},
		Account: source, SettlementMarginAsset: func(context.Context, string) (string, error) { return "USDT", nil }}
	result, err := r.ReconcileOwner(context.Background(), OwnerAutoStrategy)
	if err != nil || result.OrdersUnresolved == 0 || broker.tradeCalls == 0 {
		t.Fatalf("missing zero evidence starved REST backfill: summary=%+v calls=%d err=%v", result, broker.tradeCalls, err)
	}
	var verified int
	if err := o.Raw("SELECT rest_verified FROM futures_trade_fills WHERE order_id='1201'").QueryRow(&verified); err != nil || verified != 1 {
		t.Fatalf("failed to backfill Fill under exact-side error: rest_verified=%d err=%v", verified, err)
	}
	if got := loadSettlementPosition(t, closed.ID); got.SettlementStatus != SettlementIncomplete || got.SettlementProcessedAt != 0 {
		t.Fatalf("missing exact-side proof triggered settlement: %+v", got)
	}
	source.fail = nil
	source.perSide = map[string]AccountPosition{"BTCUSDT|LONG": {Symbol: "BTCUSDT", PositionSide: "LONG", Quantity: 0}}
	r.Account = source
	result, err = r.ReconcileOwner(context.Background(), OwnerAutoStrategy)
	if err != nil || result.OrdersUnresolved != 0 {
		t.Fatalf("explicit zero did not release settlement: %+v %v", result, err)
	}
	if got := loadSettlementPosition(t, closed.ID); got.SettlementStatus != SettlementSettled {
		t.Fatalf("valid exact zero failed settlement: %+v", got)
	}
}
