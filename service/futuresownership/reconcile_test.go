package futuresownership

import (
	"context"
	"errors"
	"fmt"
	"github.com/adshao/go-binance/v2/futures"
	"go_binance_futures/models"
	"go_binance_futures/service/internal/testtrades"
	"strings"
	"testing"
	"time"

	"github.com/beego/beego/v2/client/orm"
)

type fakeAccountPositionSource struct {
	positions []AccountPosition
	err       error
}

func (f fakeAccountPositionSource) Positions(context.Context) ([]AccountPosition, error) {
	return f.positions, f.err
}
func (f fakeAccountPositionSource) FreshPositionSide(_ context.Context, symbol, side string) (AccountPosition, error) {
	if f.err != nil {
		return AccountPosition{}, f.err
	}
	for _, row := range f.positions {
		if row.Symbol == symbol && row.PositionSide == side {
			return row, nil
		}
	}
	// Explicit zero in the test double, not an exchange claim.
	return AccountPosition{Symbol: symbol, PositionSide: side, Quantity: 0}, nil
}

func TestReconcileDoesNotClaimUnmanagedAccountPosition(t *testing.T) {
	prepareOwnershipDB(t)
	service := testService()
	reconciler := Reconciler{
		Ownership: service,
		Executor:  Executor{Ownership: service, Broker: &fakeOrderBroker{}},
		Account: fakeAccountPositionSource{positions: []AccountPosition{
			{Symbol: "BTCUSDT", PositionSide: "LONG", Quantity: 2},
		}},
	}
	summary, err := reconciler.ReconcileOwner(context.Background(), OwnerAutoStrategy)
	if err != nil {
		t.Fatal(err)
	}
	if summary.PositionsChecked != 0 || summary.OrdersChecked != 0 {
		t.Fatalf("unmanaged account state must stay observation-only: %+v", summary)
	}
	if _, err := service.GetPosition(context.Background(), OwnerAutoStrategy, "BTCUSDT", "LONG"); err != orm.ErrNoRows {
		t.Fatalf("manual account position must not be claimed, err=%v", err)
	}
}

func TestReconcileRecoversOnlyRegisteredOrderByClientID(t *testing.T) {
	prepareOwnershipDB(t)
	service := testService()
	claimOpen(t, service, OwnerAgentTrade, "ETHUSDT", "SHORT", "agt_restart", 2)
	broker := &fakeOrderBroker{lookupResult: ExchangeOrder{
		ExchangeOrderID: "901", ClientOrderID: "agt_restart", Status: "FILLED", FilledQty: 2, AveragePrice: 2200,
	}}
	reconciler := Reconciler{
		Ownership: service,
		Executor:  Executor{Ownership: service, Broker: broker},
		Account: fakeAccountPositionSource{positions: []AccountPosition{
			{Symbol: "ETHUSDT", PositionSide: "SHORT", Quantity: -2},
		}},
	}
	summary, err := reconciler.ReconcileOwner(context.Background(), OwnerAgentTrade)
	if err != nil {
		t.Fatal(err)
	}
	if summary.OrdersChecked != 1 || summary.OrdersUnresolved != 0 || broker.lookupCalls != 1 {
		t.Fatalf("registered order must reconcile exactly once: summary=%+v lookups=%d", summary, broker.lookupCalls)
	}
	position, err := service.GetPosition(context.Background(), OwnerAgentTrade, "ETHUSDT", "SHORT")
	if err != nil || position.ManagedQty != 2 {
		t.Fatalf("recovered exchange fill must restore managed position: %+v err=%v", position, err)
	}
}

func TestReconcileShrinksButNeverExpandsManagedQuantity(t *testing.T) {
	prepareOwnershipDB(t)
	service := testService()
	claimOpen(t, service, OwnerAutoStrategy, "SOLUSDT", "LONG", "aut_sol_restart", 5)
	if _, err := service.ApplyFill(context.Background(), "aut_sol_restart", 5, 150); err != nil {
		t.Fatal(err)
	}
	reconciler := Reconciler{Ownership: service, Executor: Executor{Ownership: service, Broker: &fakeOrderBroker{}}, Account: fakeAccountPositionSource{positions: []AccountPosition{{Symbol: "SOLUSDT", PositionSide: "LONG", Quantity: 8}}}}
	if _, err := reconciler.ReconcileOwner(context.Background(), OwnerAutoStrategy); err != nil {
		t.Fatal(err)
	}
	position, _ := service.GetPosition(context.Background(), OwnerAutoStrategy, "SOLUSDT", "LONG")
	if position.ManagedQty != 5 {
		t.Fatalf("manual increase must stay unmanaged, got %.8f", position.ManagedQty)
	}
	reconciler.Account = fakeAccountPositionSource{positions: []AccountPosition{{Symbol: "SOLUSDT", PositionSide: "LONG", Quantity: 3}}}
	summary, err := reconciler.ReconcileOwner(context.Background(), OwnerAutoStrategy)
	if err != nil {
		t.Fatal(err)
	}
	position, _ = service.GetPosition(context.Background(), OwnerAutoStrategy, "SOLUSDT", "LONG")
	if position.ManagedQty != 3 || summary.PositionsShrunk != 1 {
		t.Fatalf("manual reduction must shrink ownership only: position=%+v summary=%+v", position, summary)
	}
}

func TestReconcileKeepsUnknownExchangeResultFailClosed(t *testing.T) {
	prepareOwnershipDB(t)
	service := testService()
	claimOpen(t, service, OwnerNewCoinRush, "BNBUSDT", "LONG", "rush_unknown", 1)
	broker := &fakeOrderBroker{lookupErr: errors.New("exchange unavailable")}
	reconciler := Reconciler{Ownership: service, Executor: Executor{Ownership: service, Broker: broker}, Account: fakeAccountPositionSource{}}
	summary, err := reconciler.ReconcileOwner(context.Background(), OwnerNewCoinRush)
	if err != nil {
		t.Fatal(err)
	}
	if summary.OrdersUnresolved != 1 {
		t.Fatalf("unknown exchange result must remain unresolved: %+v", summary)
	}
	row, err := findTestOrder("rush_unknown")
	if err != nil || row.Status != OrderReconcile {
		t.Fatalf("unknown order must stay reconcile_required: %+v err=%v", row, err)
	}
}

type mapOrderBroker struct {
	orders      map[string]ExchangeOrder
	cancelCalls int
	cancelErr   error
}

func (b *mapOrderBroker) Submit(context.Context, OrderRequest, string) (ExchangeOrder, error) {
	return ExchangeOrder{}, errors.New("unexpected submit")
}
func (b *mapOrderBroker) Lookup(_ context.Context, _ string, clientOrderID, orderType string) (ExchangeOrder, error) {
	row, ok := b.orders[clientOrderID]
	if !ok {
		return ExchangeOrder{}, errors.New("not found")
	}
	if (strings.Contains(strings.ToUpper(orderType), "STOP") || strings.Contains(strings.ToUpper(orderType), "TAKE_PROFIT")) && row.FilledQty > 0 {
		row.ActualOrderID = row.ExchangeOrderID
	}
	return row, nil
}
func (b *mapOrderBroker) ListAccountTrades(ctx context.Context, symbol string, orderID, fromID int64, limit int) ([]*futures.AccountTrade, error) {
	return testtrades.FromTestManagedOrder(ctx, "", symbol, orderID, fromID, limit)
}
func (b *mapOrderBroker) Cancel(_ context.Context, _ string, _ int64, _ string) error {
	b.cancelCalls++
	return b.cancelErr
}

func TestReconcileCancelsSiblingProtectionAfterProtectiveFillClosesPosition(t *testing.T) {
	prepareOwnershipDB(t)
	service := testService()
	claimOpen(t, service, OwnerNoticeAutoOrder, "LINKUSDT", "LONG", "notice_entry", 1)
	if _, err := service.ApplyFill(context.Background(), "notice_entry", 1, 10); err != nil {
		t.Fatal(err)
	}
	for _, input := range []ClaimOrderInput{
		{Owner: OwnerNoticeAutoOrder, Symbol: "LINKUSDT", PositionSide: "LONG", Intent: IntentStopLoss, ClientOrderID: "notice_sl", RequestedQty: 1, OrderType: "STOP_MARKET"},
		{Owner: OwnerNoticeAutoOrder, Symbol: "LINKUSDT", PositionSide: "LONG", Intent: IntentTakeProfit, ClientOrderID: "notice_tp", RequestedQty: 1, OrderType: "TAKE_PROFIT_MARKET"},
	} {
		if _, err := service.ClaimOrder(context.Background(), input); err != nil {
			t.Fatal(err)
		}
	}
	broker := &mapOrderBroker{orders: map[string]ExchangeOrder{
		"notice_sl": {ExchangeOrderID: "101", ClientOrderID: "notice_sl", Status: "FILLED", FilledQty: 1, AveragePrice: 9},
		"notice_tp": {ExchangeOrderID: "102", ClientOrderID: "notice_tp", Status: "NEW"},
	}}
	reconciler := Reconciler{Ownership: service, Executor: Executor{Ownership: service, Broker: broker}, Account: fakeAccountPositionSource{}}
	summary, err := reconciler.ReconcileOwner(context.Background(), OwnerNoticeAutoOrder)
	if err != nil {
		t.Fatal(err)
	}
	if summary.PositionsChecked != 0 {
		// The stop fill already closed ownership while active orders were reconciled.
		t.Fatalf("protective fill should close before account-position reconcile: %+v", summary)
	}
	if broker.cancelCalls != 1 {
		t.Fatalf("remaining sibling protection must be canceled, cancel calls=%d summary=%+v", broker.cancelCalls, summary)
	}
	tp, err := findTestOrder("notice_tp")
	if err != nil || tp.Status != OrderCanceled {
		t.Fatalf("sibling protection status=%+v err=%v", tp, err)
	}
}

func TestStageReviewOversizedProtectionCanceledOnPartialClose(t *testing.T) {
	prepareOwnershipDB(t)
	s := testService()
	ctx := context.Background()
	opening := claimOpen(t, s, OwnerNoticeAutoOrder, "BTCUSDT", "LONG", "review_open4", 4)
	if _, err := s.ApplyFill(ctx, opening.ClientOrderID, 4, 100); err != nil {
		t.Fatal(err)
	}
	tp, err := s.ClaimOrder(ctx, ClaimOrderInput{Owner: OwnerNoticeAutoOrder, Symbol: "BTCUSDT", PositionSide: "LONG",
		Intent: IntentTakeProfit, ClientOrderID: "review_tp4", RequestedQty: 4, OrderType: "TAKE_PROFIT_MARKET"})
	if err != nil {
		t.Fatal(err)
	}
	if err = s.MarkOrderSubmitted(ctx, tp.ClientOrderID, "991"); err != nil {
		t.Fatal(err)
	}
	_, err = s.ClaimOrder(ctx, ClaimOrderInput{Owner: OwnerNoticeAutoOrder, Symbol: "BTCUSDT",
		PositionSide: "LONG", Intent: IntentClose, ClientOrderID: "review_close1", RequestedQty: 1, OrderType: "MARKET"})
	if err != nil {
		t.Fatal(err)
	}
	broker := &mapOrderBroker{}
	executor := Executor{Ownership: s, Broker: broker}
	_, err = executor.ApplyObservedExchange(ctx, "review_close1", ExchangeOrder{
		ExchangeOrderID: "992", FilledQty: 1, AveragePrice: 110, Status: "FILLED"})
	if err != nil {
		t.Fatal(err)
	}
	pos, err := s.GetPosition(ctx, OwnerNoticeAutoOrder, "BTCUSDT", "LONG")
	if err != nil || pos.ManagedQty != 3 {
		t.Fatalf("remaining position should be 3: %+v err=%v", pos, err)
	}
	updated, err := s.GetOrder(ctx, tp.ClientOrderID)
	if err != nil || updated.Status != OrderCanceled || updated.RequestedQty != 4 || broker.cancelCalls != 1 {
		t.Fatalf("exchange-side TP must be canceled, not locally rewritten: %+v cancels=%d err=%v", updated, broker.cancelCalls, err)
	}
}

func TestStageReviewProtectionCancelFailureMustRemainRetryable(t *testing.T) {
	prepareOwnershipDB(t)
	s := testService()
	ctx := context.Background()
	opening := claimOpen(t, s, OwnerNoticeAutoOrder, "BTCUSDT", "LONG", "review_retry_open", 4)
	if _, err := s.ApplyFill(ctx, opening.ClientOrderID, 4, 100); err != nil {
		t.Fatal(err)
	}
	tp, err := s.ClaimOrder(ctx, ClaimOrderInput{Owner: OwnerNoticeAutoOrder, Symbol: "BTCUSDT", PositionSide: "LONG",
		Intent: IntentTakeProfit, ClientOrderID: "review_retry_tp", RequestedQty: 4, OrderType: "TAKE_PROFIT_MARKET"})
	if err != nil {
		t.Fatal(err)
	}
	if err = s.MarkOrderSubmitted(ctx, tp.ClientOrderID, "993"); err != nil {
		t.Fatal(err)
	}
	_, err = s.ClaimOrder(ctx, ClaimOrderInput{Owner: OwnerNoticeAutoOrder, Symbol: "BTCUSDT",
		PositionSide: "LONG", Intent: IntentClose, ClientOrderID: "review_retry_close", RequestedQty: 1, OrderType: "MARKET"})
	if err != nil {
		t.Fatal(err)
	}
	broker := &mapOrderBroker{cancelErr: errors.New("exchange cancel unavailable")}
	executor := Executor{Ownership: s, Broker: broker}
	_, err = executor.ApplyObservedExchange(ctx, "review_retry_close", ExchangeOrder{
		ExchangeOrderID: "994", FilledQty: 1, AveragePrice: 110, Status: "FILLED"})
	if err == nil {
		t.Fatal("remote protection cancel failure must surface")
	}
	still, err := s.GetOrder(ctx, tp.ClientOrderID)
	if err != nil || still.Status == OrderCanceled || still.RequestedQty != 4 {
		t.Fatalf("unknown exchange outcome must not fake a local cancel: %+v err=%v", still, err)
	}
	broker.cancelErr = nil
	if err = executor.cancelExcessProtections(ctx, *still.PositionID); err != nil {
		t.Fatal(err)
	}
	again, err := s.GetOrder(ctx, tp.ClientOrderID)
	if err != nil || again.Status != OrderCanceled {
		t.Fatalf("protection retry failed: %+v %v", again, err)
	}
}

func TestStageReviewEmptySnapshotUnresolvedOrderCannotProveFlat(t *testing.T) {
	prepareOwnershipDB(t)
	s := testService()
	ctx := context.Background()
	open := claimOpen(t, s, OwnerAutoStrategy, "SOLUSDT", "LONG", "review_empty_open", 2)
	if _, err := s.ApplyFill(ctx, open.ClientOrderID, 1, 100); err != nil {
		t.Fatal(err)
	}
	broker := &fakeOrderBroker{lookupErr: errors.New("unconfirmed order at exchange")}
	r := Reconciler{Ownership: s, Executor: Executor{Ownership: s, Broker: broker},
		Account: fakeAccountPositionSource{positions: nil}}
	summary, err := r.ReconcileOwner(ctx, OwnerAutoStrategy)
	if err != nil || summary.OrdersUnresolved == 0 {
		t.Fatalf("pending unresolved order should block zero assumption: %+v %v", summary, err)
	}
	pos, err := s.GetPosition(ctx, OwnerAutoStrategy, "SOLUSDT", "LONG")
	if err != nil || pos.Status == PositionClosed || pos.ManagedQty != 1 {
		t.Fatalf("suspicious empty account snapshot destroyed owned position: %+v %v", pos, err)
	}
}

func TestStageReviewOldIncompleteSettlementSurfacesAlarmCounters(t *testing.T) {
	prepareOwnershipDB(t)
	base := testService()
	ctx := context.Background()
	open := claimOpen(t, base, OwnerAutoStrategy, "XRPUSDT", "LONG", "review_aged_open", 1)
	if _, err := base.ApplyFill(ctx, open.ClientOrderID, 1, 100); err != nil {
		t.Fatal(err)
	}
	closed, err := base.ReconcilePosition(ctx, OwnerAutoStrategy, "XRPUSDT", "LONG", 0)
	if err != nil || closed.SettlementStatus != SettlementIncomplete {
		t.Fatalf("expected incomplete cycle: %+v %v", closed, err)
	}
	advanced := base
	advanced.Now = func() time.Time { return time.UnixMilli(closed.ClosedAt + 16*60*1000).UTC() }
	r := Reconciler{Ownership: advanced, Executor: Executor{Ownership: advanced, Broker: &fakeOrderBroker{}},
		Account:               fakeAccountPositionSource{positions: nil},
		SettlementMarginAsset: func(context.Context, string) (string, error) { return "USDT", nil }}
	summary, err := r.ReconcileOwner(ctx, OwnerAutoStrategy)
	if err != nil || summary.IncompleteAged != 1 || summary.OldestIncompleteAgeSec < 15*60 {
		t.Fatalf("stale incomplete ledger not surfaced to monitoring: %+v %v", summary, err)
	}
	unchanged := loadSettlementPosition(t, closed.ID)
	if unchanged.SettlementStatus != SettlementIncomplete || unchanged.SettlementProcessedAt != 0 {
		t.Fatalf("stale alarm must not manufacture a settlement: %+v", unchanged)
	}
}

type reviewPartialAccountSnapshot struct {
	bulk    []AccountPosition
	perSide map[string]AccountPosition
	fail    map[string]bool
}

func (p reviewPartialAccountSnapshot) Positions(context.Context) ([]AccountPosition, error) {
	return p.bulk, nil
}
func (p reviewPartialAccountSnapshot) FreshPositionSide(_ context.Context, symbol, side string) (AccountPosition, error) {
	k := positionKey(symbol, side)
	if p.fail[k] {
		return AccountPosition{}, fmt.Errorf("targeted REST snapshot missing %s", k)
	}
	row, ok := p.perSide[k]
	if !ok {
		return AccountPosition{}, fmt.Errorf("no explicit position side %s", k)
	}
	return row, nil
}
func TestStageR2PartiallyMissingBulkSnapshotRequiresExactRESTProof(t *testing.T) {
	prepareOwnershipDB(t)
	s := testService()
	ctx := context.Background()
	for _, it := range []struct{ id, symbol string }{{"r2_snap_btc", "BTCUSDT"}, {"r2_snap_eth", "ETHUSDT"}} {
		open := claimOpen(t, s, OwnerAutoStrategy, it.symbol, "LONG", it.id, 1)
		if _, err := s.ApplyFill(ctx, open.ClientOrderID, 1, 100); err != nil {
			t.Fatal(err)
		}
	}
	source := reviewPartialAccountSnapshot{
		bulk: []AccountPosition{{Symbol: "BTCUSDT", PositionSide: "LONG", Quantity: 1}},
		fail: map[string]bool{"ETHUSDT|LONG": true},
	}
	r := Reconciler{Ownership: s, Executor: Executor{Ownership: s, Broker: &fakeOrderBroker{}}, Account: source}
	result, err := r.ReconcileOwner(ctx, OwnerAutoStrategy)
	if err != nil || result.OrdersUnresolved == 0 {
		t.Fatalf("partial REST failure ignored: %+v %v", result, err)
	}
	eth, err := s.GetPosition(ctx, OwnerAutoStrategy, "ETHUSDT", "LONG")
	if err != nil || eth.Status == PositionClosed || eth.ManagedQty != 1 {
		t.Fatalf("ETH missing from bulk was wrongly closed: %+v %v", eth, err)
	}
	source.fail = nil
	source.perSide = map[string]AccountPosition{"ETHUSDT|LONG": {Symbol: "ETHUSDT", PositionSide: "LONG", Quantity: 0}}
	r.Account = source
	result, err = r.ReconcileOwner(ctx, OwnerAutoStrategy)
	if err != nil || result.OrdersUnresolved != 0 {
		t.Fatalf("exact-zero REST side check failed: %+v %v", result, err)
	}
	eth, err = s.GetPosition(ctx, OwnerAutoStrategy, "ETHUSDT", "LONG")
	if err != nil || eth.Status != PositionReconcileRequired {
		t.Fatalf("first exact-zero confirmation must start two-step closure: %+v %v", eth, err)
	}
}

func TestStageR2IncompleteAlarmThrottleStorageIsBounded(t *testing.T) {
	incompleteAlertMu.Lock()
	incompleteAlertAt = make(map[string]int64)
	for i := 0; i < 4200; i++ {
		incompleteAlertAt[fmt.Sprintf("past/%d", i)] = 10
	}
	incompleteAlertMu.Unlock()
	pos := models.FuturesManagedPosition{AccountID: "main", ID: 999, Owner: OwnerAgentTrade,
		Symbol: "ETHUSDT", PositionSide: "LONG", ClosedAt: 1000}
	result := ReconcileSummary{}
	alarmIncompletePosition(pos, 1_800_000_000_000, "test", &result)
	incompleteAlertMu.Lock()
	n := len(incompleteAlertAt)
	incompleteAlertMu.Unlock()
	if n > 4096 || result.IncompleteAged != 1 {
		t.Fatalf("unbounded alarm cache n=%d summary=%+v", n, result)
	}
	clearIncompleteAlarm(pos.AccountID, pos.ID)
	incompleteAlertMu.Lock()
	_, still := incompleteAlertAt["main/999"]
	incompleteAlertMu.Unlock()
	if still {
		t.Fatal("settled key not deleted from alarm cache")
	}
}

func TestStageR2FakeRESTSupportsNonzeroMoneyWithoutLeakage(t *testing.T) {
	prepareOwnershipDB(t)
	ctx := context.Background()
	s := testService()
	open := claimOpen(t, s, OwnerAutoStrategy, "BTCUSDT", "LONG", "r2_fake_money", 1)
	testtrades.SetAmountsForExchangeOrder(99102, "0.25", "USDT", "-1.75")
	e := Executor{Ownership: s, Broker: &fakeOrderBroker{}}
	if _, err := e.ApplyObservedExchange(ctx, open.ClientOrderID, ExchangeOrder{
		ExchangeOrderID: "99102", Status: "FILLED", FilledQty: 1, AveragePrice: 100}); err != nil {
		t.Fatal(err)
	}
	var commission, pnl, asset string
	var verified int
	if err := orm.NewOrm().Raw(`SELECT CAST(commission AS TEXT), CAST(realized_pnl AS TEXT),
        commission_asset, rest_verified FROM futures_trade_fills WHERE order_id='99102'`).
		QueryRow(&commission, &pnl, &asset, &verified); err != nil {
		t.Fatal(err)
	}
	if commission != "0.25" || pnl != "-1.75" || asset != "USDT" || verified != 1 {
		t.Fatalf("injected REST money did not survive: fee=%q pnl=%q asset=%q verified=%d", commission, pnl, asset, verified)
	}
	testtrades.Reset()
	// The helper reset must clear prior account/order money, not just fills.
	probe, err := testtrades.FromTestManagedOrder(ctx, "main", "BTCUSDT", 99102, 0, 1000)
	if err != nil {
		t.Fatal(err)
	}
	if len(probe) != 1 || probe[0].Commission != "0" || probe[0].RealizedPnl != "0" {
		t.Fatalf("test trade financial state leaked across runs: %+v", probe)
	}
}

// Deliberately lacks ExactPositionSideSource. No source can infer zero from
// the absence of a side in the bulk REST payload.
type stageR3NoExactSnapshot struct{ rows []AccountPosition }

func (s stageR3NoExactSnapshot) Positions(context.Context) ([]AccountPosition, error) {
	return s.rows, nil
}

func TestStageR3NoExactSideProviderCannotCloseManagedPosition(t *testing.T) {
	prepareOwnershipDB(t)
	svc := testService()
	open := claimOpen(t, svc, OwnerAutoStrategy, "ETHUSDT", "LONG", "r3_nonexact", 1)
	if _, err := svc.ApplyFill(context.Background(), open.ClientOrderID, 1, 100); err != nil {
		t.Fatal(err)
	}
	r := Reconciler{Ownership: svc, Executor: Executor{Ownership: svc, Broker: &fakeOrderBroker{}},
		Account: stageR3NoExactSnapshot{}}
	summary, err := r.ReconcileOwner(context.Background(), OwnerAutoStrategy)
	if err != nil || summary.OrdersUnresolved < 1 {
		t.Fatalf("nonexact source fail-open: %+v %v", summary, err)
	}
	pos, err := svc.GetPosition(context.Background(), OwnerAutoStrategy, "ETHUSDT", "LONG")
	if err != nil || pos.Status == PositionClosed || pos.ManagedQty != 1 {
		t.Fatalf("nonexact source closed position: %+v %v", pos, err)
	}
}

// R3-N6: rotate a sorted budget instead of sampling random map iteration or
// permanently starving all keys alphabetically after the first six.
type stageR3TracingExactSource struct{ seen map[string]int }

func (s *stageR3TracingExactSource) Positions(context.Context) ([]AccountPosition, error) {
	return nil, nil
}
func (s *stageR3TracingExactSource) FreshPositionSide(_ context.Context, symbol, side string) (AccountPosition, error) {
	key := positionKey(symbol, side)
	if s.seen == nil {
		s.seen = make(map[string]int)
	}
	s.seen[key]++
	return AccountPosition{Symbol: symbol, PositionSide: side, Quantity: 1}, nil
}
func TestStageR3ExactSnapshotRateBudgetRotatesAcrossEightSymbols(t *testing.T) {
	prepareOwnershipDB(t)
	clock := time.UnixMilli(1_800_000_000_000).UTC()
	svc := testService()
	svc.Now = func() time.Time { return clock }
	ctx := context.Background()
	for i := 0; i < 8; i++ {
		symbol := fmt.Sprintf("R3%02dUSDT", i)
		id := fmt.Sprintf("r3_budget_%d", i)
		order := claimOpen(t, svc, OwnerAutoStrategy, symbol, "LONG", id, 1)
		if _, err := svc.ApplyFill(ctx, order.ClientOrderID, 1, 100); err != nil {
			t.Fatal(err)
		}
	}
	source := &stageR3TracingExactSource{}
	r := Reconciler{Ownership: svc, Executor: Executor{Ownership: svc, Broker: &fakeOrderBroker{}}, Account: source}
	first, err := r.ReconcileOwner(ctx, OwnerAutoStrategy)
	if err != nil || first.OrdersUnresolved < 2 || len(source.seen) != 6 {
		t.Fatalf("first six-call side verification budget: %+v seen=%d err=%v", first, len(source.seen), err)
	}
	clock = clock.Add(2 * time.Minute)
	second, err := r.ReconcileOwner(ctx, OwnerAutoStrategy)
	if err != nil || second.OrdersUnresolved < 2 || len(source.seen) != 8 {
		t.Fatalf("rotating snapshot checks starved keys: %+v seen=%d err=%v", second, len(source.seen), err)
	}
}

// A wrapper that returns an explicit bulk zero/reduction but lacks the
// side-scoped fresh REST contract must be fail-closed, not just missing-key
// snapshots. This is the production source interface security boundary.
func TestStageR3NonExactSnapshotCannotShrinkEvenWithPresentKey(t *testing.T) {
	prepareOwnershipDB(t)
	svc := testService()
	ctx := context.Background()
	opening := claimOpen(t, svc, OwnerAutoStrategy, "ADAUSDT", "LONG", "r3_nonexact_present", 1)
	if _, err := svc.ApplyFill(ctx, opening.ClientOrderID, 1, 100); err != nil {
		t.Fatal(err)
	}
	for _, qty := range []float64{0.25, 0} {
		r := Reconciler{Ownership: svc, Executor: Executor{Ownership: svc, Broker: &fakeOrderBroker{}},
			Account: stageR3NoExactSnapshot{rows: []AccountPosition{{Symbol: "ADAUSDT", PositionSide: "LONG", Quantity: qty}}}}
		summary, err := r.ReconcileOwner(ctx, OwnerAutoStrategy)
		if err != nil || summary.OrdersUnresolved == 0 {
			t.Fatalf("non-exact snapshot accepted reduction %.2f: summary=%+v err=%v", qty, summary, err)
		}
		pos, err := svc.GetPosition(ctx, OwnerAutoStrategy, "ADAUSDT", "LONG")
		if err != nil || pos.ManagedQty != 1 || pos.Status == PositionClosed {
			t.Fatalf("non-exact %.2f incorrectly shrunk/closed ownership: %+v %v", qty, pos, err)
		}
	}
}
