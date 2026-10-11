package leadaccount

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/adshao/go-binance/v2/futures"
	"github.com/beego/beego/v2/client/orm"
	_ "github.com/mattn/go-sqlite3"
	binance "go_binance_futures/feature/api/binance"
	"go_binance_futures/models"
	"go_binance_futures/service/futuresownership"
)

func prepareLeadExecutionDB(t *testing.T) {
	t.Helper()
	_ = orm.RegisterDriver("sqlite3", orm.DRSqlite)
	if _, err := orm.GetDB("default"); err != nil {
		if err := orm.RegisterDataBase("default", "sqlite3", "file:lead_execution_stage43?mode=memory&cache=shared"); err != nil {
			t.Fatal(err)
		}
		orm.RegisterModel(new(models.FuturesManagedPosition), new(models.FuturesManagedOrder))
		if err := orm.RunSyncdb("default", false, false); err != nil {
			t.Fatal(err)
		}
	}
	db := orm.NewOrm()
	for _, table := range []string{"futures_managed_orders", "futures_managed_positions"} {
		if _, err := db.Raw("DELETE FROM " + table).Exec(); err != nil {
			t.Fatal(err)
		}
	}
}

type stage43Broker struct {
	mu           sync.Mutex
	submits      int
	lookups      int
	result       futuresownership.ExchangeOrder
	err          error
	lookupResult futuresownership.ExchangeOrder
	lookupErr    error
}

func (b *stage43Broker) Submit(_ context.Context, r futuresownership.OrderRequest, clientID string) (futuresownership.ExchangeOrder, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.submits++
	if b.err != nil {
		return futuresownership.ExchangeOrder{}, b.err
	}
	v := b.result
	v.ClientOrderID = clientID
	return v, nil
}
func (b *stage43Broker) Lookup(_ context.Context, _, _, _ string) (futuresownership.ExchangeOrder, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.lookups++
	return b.lookupResult, b.lookupErr
}
func (b *stage43Broker) Cancel(context.Context, string, int64, string) error { return nil }
func stage43Adapter(t *testing.T, allowInTests bool) (*LeadExecutionAdapter, *stage43Broker) {
	t.Helper()
	client, err := binance.NewAccountClient(binance.LeadAccountID, futures.NewClient("lead-stage43-fake", "fake-secret"))
	if err != nil {
		t.Fatal(err)
	}
	a, err := NewLeadExecutionAdapter(client, riskSymbols(t))
	if err != nil {
		t.Fatal(err)
	}
	if a.ownership.Ownership.AccountID != "lead" {
		t.Fatal("ownership not bound to Lead")
	}
	broker := &stage43Broker{result: futuresownership.ExchangeOrder{ExchangeOrderID: "900", Status: "FILLED", FilledQty: 1, AveragePrice: 100}}
	a.ownership.Broker = broker
	fake := sampleLeadEvidenceReader()
	fake.positions = nil
	fake.orders = nil
	a.source.Reader = fake
	_, closeSymbol, _, _ := stage43RuleCase()
	a.rules.Reader = &fakeLeadRulesReader{exchange: &futures.ExchangeInfo{Symbols: []futures.Symbol{closeSymbol}}}
	a.simulateRules = func(RiskOpenOrder) (VerifiedOrderRules, error) {
		return VerifiedOrderRules{MinNotionalUSDT: 5, MaxNotionalUSDT: 10000, MaxLeverage: 20}, nil
	}
	if allowInTests {
		// These fields are private and only a same-package test can set them.
		// Production has neither an Enable method nor a Stage7 authorization API.
		a.risk.mu.Lock()
		a.risk.state = riskRunState{Enabled: true, AllowNewOpens: true, PortfolioBound: true, WSHealthy: true, Stage7Authorized: true, ReadOnlyVerified: true}
		a.risk.limits = RiskLimits{MaxOrderNotionalUSDT: 200, MaxTotalNotionalUSDT: 500, MaxDailyRealizedLossUSDT: 10, MaxPositions: 5, MaxLosingPositions: 2}
		a.risk.mu.Unlock()
		a.simulateEvidence = func(s *RiskSnapshot) {
			s.Reconciled = true
			s.UnknownOrders = false
		}
	}
	return a, broker
}
func stage43Request(side, kind, id string) (futuresownership.OrderRequest, RiskOpenOrder) {
	orderSide := "BUY"
	if side == "SHORT" {
		orderSide = "SELL"
	}
	price := 0.0
	if kind == "LIMIT" {
		price = 100
	}
	request := futuresownership.OrderRequest{Owner: futuresownership.OwnerAutoStrategy, Symbol: "BTCUSDT", PositionSide: side, Side: orderSide,
		Intent: futuresownership.IntentOpen, OrderType: kind, Quantity: 1, Price: price, ClientOrderID: id}
	check := RiskOpenOrder{AccountID: binance.LeadAccountID, Symbol: "BTCUSDT", Side: side, OrderType: kind, Quantity: 1, ReferencePrice: 100,
		Leverage: 4, MarginType: "CROSSED", ConfiguredUSDT: 27, BinanceMinNotionalUSDT: 5, BinanceMaxNotionalUSDT: 10000, BinanceMaxLeverage: 20,
		PriceAndQuantityValidated: true, MarginModeVerified: true}
	return request, check
}
func TestStage43ProductionAdapterDefaultDeniedBeforeDBAndExchange(t *testing.T) {
	a, broker := stage43Adapter(t, false)
	req, check := stage43Request("LONG", "MARKET", "lead_no_permission")
	_, err := a.ExecuteOpen(context.Background(), req, check)
	if reader, ok := a.source.Reader.(*fakeLeadSnapshotReader); ok && len(reader.calls) != 0 {
		t.Fatalf("disabled Lead adapter contacted private APIs: %v", reader.calls)
	}
	if !errors.Is(err, ErrLeadOpenBlocked) || broker.submits != 0 {
		t.Fatalf("unsafe adapter executed without gate: %v submits=%d", err, broker.submits)
	}
}
func TestStage43RealExecutorRejectsMainClient(t *testing.T) {
	main, err := binance.NewAccountClient(binance.MainAccountID, futures.NewClient("main", "secret"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := NewLeadExecutionAdapter(main, riskSymbols(t)); err == nil {
		t.Fatal("main client accepted by Lead adapter")
	}
	if _, err := NewLeadExecutionAdapter(nil, riskSymbols(t)); err == nil {
		t.Fatal("nil client accepted")
	}
}
func TestStage43MockExecutesThenReservesUntilReconcile(t *testing.T) {
	prepareLeadExecutionDB(t)
	for _, tc := range []struct{ name, side, kind string }{
		{"long_market", "LONG", "MARKET"}, {"short_limit", "SHORT", "LIMIT"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			prepareLeadExecutionDB(t)
			a, broker := stage43Adapter(t, true)
			req, check := stage43Request(tc.side, tc.kind, "stage43_"+tc.name)
			exchange, err := a.ExecuteOpen(context.Background(), req, check)
			if err != nil || exchange.FilledQty != 1 || broker.submits != 1 {
				t.Fatalf("mock submit failed: result=%+v err=%v count=%d", exchange, err, broker.submits)
			}
			row, err := a.ownership.Ownership.GetOrder(context.Background(), req.ClientOrderID)
			if err != nil || row.AccountID != "lead" || row.Status != futuresownership.OrderFilled {
				t.Fatalf("persisted ownership incorrect %+v error=%v", row, err)
			}
			position, err := a.ownership.Ownership.GetPosition(context.Background(), futuresownership.OwnerAutoStrategy, "BTCUSDT", tc.side)
			if err != nil || position.ManagedQty != 1 || position.AccountID != "lead" {
				t.Fatalf("filled qty ownership incorrect %+v err=%v", position, err)
			}
			_, err = a.ExecuteOpen(context.Background(), req, check)
			if !errors.Is(err, ErrLeadOpenBlocked) || !errors.Is(err, ErrLeadPendingReconcile) || broker.submits != 1 {
				t.Fatalf("repeat open lost pending category or bypassed reconciliation err=%v submits=%d", err, broker.submits)
			}
			decision, err := a.CheckOpen(context.Background(), check)
			if !errors.Is(err, ErrLeadPendingReconcile) || !riskReasonsContain(decision, "lead_pending_reconcile") {
				t.Fatalf("pending status unblocked: %+v %v", decision, err)
			}
		})
	}
}
func TestStage43UnknownSubmissionKeepsPendingAndNoBlindRetry(t *testing.T) {
	prepareLeadExecutionDB(t)
	a, broker := stage43Adapter(t, true)
	broker.err = errors.New("simulated network dropped after request")
	broker.lookupErr = errors.New("lookup temporarily unavailable")
	req, check := stage43Request("LONG", "MARKET", "stage43_unknown_1")
	if _, err := a.ExecuteOpen(context.Background(), req, check); err == nil {
		t.Fatal("unknown submit not returned")
	}
	if broker.submits != 1 || broker.lookups != 1 {
		t.Fatalf("unknown result was not reconciled: %+v", broker)
	}
	row, err := a.ownership.Ownership.GetOrder(context.Background(), req.ClientOrderID)
	if err != nil || row.Status != futuresownership.OrderReconcile || row.AccountID != "lead" {
		t.Fatalf("unknown result not retained: %+v %v", row, err)
	}
	if _, err := a.ExecuteOpen(context.Background(), req, check); !errors.Is(err, ErrLeadOpenBlocked) {
		t.Fatalf("second submit not blocked: %v", err)
	}
	if broker.submits != 1 {
		t.Fatalf("duplicate unknown submit sent: %d", broker.submits)
	}
}
func TestStage43ConcurrencySerializesLeadAttempts(t *testing.T) {
	prepareLeadExecutionDB(t)
	a, broker := stage43Adapter(t, true)
	req, check := stage43Request("LONG", "MARKET", "stage43_concurrent")
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); _, _ = a.ExecuteOpen(context.Background(), req, check) }()
	}
	wg.Wait()
	if broker.submits != 1 {
		t.Fatalf("concurrent Lead submitted %d orders", broker.submits)
	}
}
func TestStage43IncorrectOrderRequestDoesNotReserveOrSubmit(t *testing.T) {
	prepareLeadExecutionDB(t)
	for _, tc := range []struct {
		name string
		bad  func(*futuresownership.OrderRequest, *RiskOpenOrder)
	}{
		{"wrong_side", func(r *futuresownership.OrderRequest, _ *RiskOpenOrder) { r.Side = "SELL" }},
		{"wrong_quantity", func(r *futuresownership.OrderRequest, _ *RiskOpenOrder) { r.Quantity = 2 }},
		{"wrong_symbol", func(r *futuresownership.OrderRequest, _ *RiskOpenOrder) { r.Symbol = "ETHUSDT" }},
		{"wrong_price", func(r *futuresownership.OrderRequest, _ *RiskOpenOrder) { r.Price = 10 }},
		{"wrong_owner", func(r *futuresownership.OrderRequest, _ *RiskOpenOrder) { r.Owner = futuresownership.OwnerAgentTrade }},
		{"wrong_intent", func(r *futuresownership.OrderRequest, _ *RiskOpenOrder) { r.Intent = futuresownership.IntentClose }},
		{"wrong_account", func(_ *futuresownership.OrderRequest, c *RiskOpenOrder) { c.AccountID = binance.MainAccountID }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, broker := stage43Adapter(t, true)
			req, check := stage43Request("LONG", "LIMIT", fmt.Sprintf("stage43_bad_%s", tc.name))
			tc.bad(&req, &check)
			_, err := a.ExecuteOpen(context.Background(), req, check)
			if !errors.Is(err, ErrLeadOpenBlocked) || broker.submits != 0 {
				t.Fatalf("unsafe order permitted %s err=%v count=%d", tc.name, err, broker.submits)
			}
		})
	}
}
func TestStage43CancelDoesNotProduceFuturesOrders(t *testing.T) {
	a, broker := stage43Adapter(t, true)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	req, check := stage43Request("SHORT", "MARKET", "stage43_cancel")
	if _, err := a.ExecuteOpen(ctx, req, check); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancel=%v", err)
	}
	if broker.submits != 0 {
		t.Fatal("cancelled request submitted")
	}
	_ = time.Second
}
