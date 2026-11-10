package spotstrategy

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/beego/beego/v2/client/orm"
	"github.com/shopspring/decimal"
	"go_binance_futures/models"
)

type fakeS2Client struct {
	sync.Mutex
	order                                     OrderSnapshot
	trades                                    []Trade
	calls, queries, canceled                  int
	createErr, queryErr, tradesErr, cancelErr error
	instrumentErr, balancesErr, openErr       error
	mutateOnCancel                            []Trade
	unmanaged                                 []OrderSnapshot
	free                                      string
	baseFree                                  string
	instrument                                Instrument
}

func newFakeS2() *fakeS2Client {
	return &fakeS2Client{
		free: "10000", baseFree: "0", instrument: Instrument{Symbol: "BTCUSDT", BaseAsset: "BTC", QuoteAsset: "USDT", Status: "TRADING", SpotAllowed: true,
			LotMin: "1", LotMax: "100", LotStep: "1", TickSize: "1", MinNotional: "10"},
	}
}
func (f *fakeS2Client) Balances(ctx context.Context, account string) (map[string]Balance, error) {
	if f.balancesErr != nil {
		return nil, f.balancesErr
	}
	if account != AccountMain {
		return nil, errors.New("wrong account")
	}
	return map[string]Balance{"BTC": {Free: f.baseFree, Locked: "0"}, "USDT": {Free: f.free, Locked: "0"}}, nil
}
func (f *fakeS2Client) Instrument(ctx context.Context, sym string) (Instrument, error) {
	if f.instrumentErr != nil {
		return Instrument{}, f.instrumentErr
	}
	return f.instrument, nil
}
func (f *fakeS2Client) OpenOrders(ctx context.Context, account, sym string) ([]OrderSnapshot, error) {
	f.Lock()
	defer f.Unlock()
	if f.openErr != nil {
		return nil, f.openErr
	}
	return append([]OrderSnapshot(nil), f.unmanaged...), nil
}
func (f *fakeS2Client) QueryOrder(ctx context.Context, account, sym, clientID, orderID string) (OrderSnapshot, error) {
	f.Lock()
	defer f.Unlock()
	f.queries++
	if f.queryErr != nil {
		return OrderSnapshot{}, f.queryErr
	}
	if f.order.OrderID == "" || f.order.Symbol != sym || (orderID != "" && orderID != f.order.OrderID) ||
		(orderID == "" && clientID != f.order.ClientOrderID) {
		return OrderSnapshot{}, errors.New("Binance -2013 unknown order")
	}
	return f.order, nil
}
func (f *fakeS2Client) Trades(ctx context.Context, account, sym, orderID string, fromID int64, limit int) ([]Trade, error) {
	f.Lock()
	defer f.Unlock()
	if f.tradesErr != nil {
		return nil, f.tradesErr
	}
	items := append([]Trade(nil), f.trades...)
	sort.Slice(items, func(i, j int) bool {
		l, _ := strconv.ParseInt(items[i].TradeID, 10, 64)
		r, _ := strconv.ParseInt(items[j].TradeID, 10, 64)
		return l < r
	})
	out := []Trade{}
	for _, v := range items {
		n, _ := strconv.ParseInt(v.TradeID, 10, 64)
		if n >= fromID {
			out = append(out, v)
		}
		if len(out) == limit {
			break
		}
	}
	return out, nil
}
func (f *fakeS2Client) Place(ctx context.Context, req OrderRequest) (OrderSnapshot, error) {
	f.Lock()
	defer f.Unlock()
	f.calls++
	f.order = OrderSnapshot{Symbol: req.Symbol, OrderID: "901", ClientOrderID: req.ClientOrderID, Side: req.Side,
		Status: "NEW", RequestedQty: req.Quantity, ExecutedQty: "0", ExecutedQuoteQty: "0"}
	return f.order, f.createErr
}
func (f *fakeS2Client) Cancel(ctx context.Context, account, sym, clientID string) error {
	f.Lock()
	defer f.Unlock()
	f.canceled++
	if f.cancelErr != nil {
		return f.cancelErr
	}
	f.trades = append(f.trades, f.mutateOnCancel...)
	f.order.Status = "CANCELED"
	total, quote := decimal.Zero, decimal.Zero
	for _, tr := range f.trades {
		n, _ := decimal.NewFromString(tr.Qty)
		q, _ := decimal.NewFromString(tr.QuoteQty)
		total = total.Add(n)
		quote = quote.Add(q)
	}
	f.order.ExecutedQty = total.String()
	f.order.ExecutedQuoteQty = quote.String()
	// Keep origClientOrderID stable in the happy-path fake. A changed ID
	// must now be quarantined, not silently accepted by orderId alone.
	return nil
}
func prepareS2(t *testing.T) (*Engine, *fakeS2Client) {
	t.Helper()
	setupSpotS1(t)
	o := orm.NewOrm()
	if _, err := o.Raw("UPDATE spot_symbols SET strategy_trade_enabled=1,strategy_budget_quote='2000' WHERE symbol='BTCUSDT'").Exec(); err != nil {
		t.Fatal(err)
	}
	db, err := orm.GetDB("default")
	if err != nil {
		t.Fatal(err)
	}
	f := newFakeS2()
	return NewMockEngine(db, f), f
}
func requestS2(key string) OrderRequest {
	return OrderRequest{AccountID: AccountMain, Owner: OwnerAuto, Symbol: "BTCUSDT", Side: "BUY", Type: "LIMIT",
		Quantity: "10", Price: "100", SignalKey: key}
}
func mustPrepareSubmitS2(t *testing.T, e *Engine, key string) LocalOrder {
	t.Helper()
	order, err := e.Prepare(context.Background(), requestS2(key))
	if err != nil {
		t.Fatal("prepare:", err)
	}
	if order.ClientOrderID == "" || order.Status != StateReserved {
		t.Fatalf("invalid reserved order: %+v", order)
	}
	out, err := e.Submit(context.Background(), order.ID, "100", "100")
	if err != nil {
		t.Fatal("submit:", err)
	}
	return out
}
func tradeS2(id int64, qty, price, quote, fee, asset string) Trade {
	return Trade{Symbol: "BTCUSDT", OrderID: "901", TradeID: strconv.FormatInt(id, 10), Side: "BUY", Qty: qty,
		Price: price, QuoteQty: quote, Commission: fee, CommissionAsset: asset, Time: time.Now().UnixMilli()}
}
func countTableS2(t *testing.T, table string) int {
	t.Helper()
	db, _ := orm.GetDB("default")
	var n int
	if err := db.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestS2T01_ACKDoesNotCreateFillOrHolding(t *testing.T) {
	e, _ := prepareS2(t)
	got := mustPrepareSubmitS2(t, e, "t2-01")
	if got.Status != StateNew || got.FilledQty != "0" || countTableS2(t, "spot_strategy_fills") != 0 ||
		countTableS2(t, "spot_strategy_holdings") != 0 {
		t.Fatalf("ACK was mistaken for Fill: %+v", got)
	}
}
func TestS2T02AndT08_PartialFillCancelKeepsRealFills(t *testing.T) {
	e, f := prepareS2(t)
	got := mustPrepareSubmitS2(t, e, "t2-02")
	f.Lock()
	f.trades = []Trade{tradeS2(11, "2", "100", "200", "0.001", "BTC")}
	f.order.Status = "PARTIALLY_FILLED"
	f.order.ExecutedQty = "2"
	f.order.ExecutedQuoteQty = "200"
	f.mutateOnCancel = []Trade{tradeS2(12, "2", "120", "240", "0.002", "BNB")}
	f.Unlock()
	before, err := e.Reconcile(context.Background(), got.ID)
	if err != nil || before.Status != StatePartial || before.FilledQty != "2" {
		t.Fatalf("partial %+v err=%v", before, err)
	}
	final, err := e.CancelMock(context.Background(), got.ID)
	if err != nil || final.Status != StateCanceled || final.FilledQty != "4" || final.ExecutedQuoteQty != "440" || final.ReservedQty != "0" {
		t.Fatalf("canceled partial %+v err=%v", final, err)
	}
	if countTableS2(t, "spot_strategy_fills") != 2 || countTableS2(t, "spot_strategy_holdings") != 0 {
		t.Fatal("fill or holdings count incorrect")
	}
	recorded, readErr := e.ListFills(context.Background(), got.ID)
	if readErr != nil || len(recorded) != 2 || recorded[0].CommissionAsset != "BTC" || recorded[1].CommissionAsset != "BNB" ||
		recorded[0].RESTVerified || recorded[1].RESTVerified {
		t.Fatalf("incorrect fee asset / fake REST origin: %+v %v", recorded, readErr)
	}
	if _, err = e.Reconcile(context.Background(), got.ID); err != nil {
		t.Fatal("terminal recovery:", err)
	}
	if countTableS2(t, "spot_strategy_fills") != 2 {
		t.Fatal("terminal replay duplicated fills")
	}
	db, _ := orm.GetDB("default")
	var missing int
	if err = db.QueryRow("SELECT COUNT(*) FROM spot_strategy_fills WHERE rest_verified<>0").Scan(&missing); err != nil || missing != 0 {
		t.Fatalf("mock fills masqueraded as real REST verified: %d %v", missing, err)
	}
}
func TestS2T03_DuplicateTradeIdHundredTimes(t *testing.T) {
	e, f := prepareS2(t)
	got := mustPrepareSubmitS2(t, e, "t2-03")
	tr := tradeS2(44, "1", "100", "100", "0.005", "USDT")
	f.Lock()
	for i := 0; i < 100; i++ {
		f.trades = append(f.trades, tr)
	}
	f.order.Status = "PARTIALLY_FILLED"
	f.order.ExecutedQty = "1"
	f.order.ExecutedQuoteQty = "100"
	f.Unlock()
	for i := 0; i < 5; i++ {
		order, err := e.Reconcile(context.Background(), got.ID)
		if err != nil || order.FilledQty != "1" {
			t.Fatalf("dup id: %+v %v", order, err)
		}
	}
	if countTableS2(t, "spot_strategy_fills") != 1 {
		t.Fatalf("dup fill count %d", countTableS2(t, "spot_strategy_fills"))
	}
}
func TestS2T04T05_TimeoutAcceptedAndMissingNeverResubmit(t *testing.T) {
	e, f := prepareS2(t)
	f.createErr = errors.New("transport TLS timeout")
	order, err := e.Prepare(context.Background(), requestS2("t2-04"))
	if err != nil {
		t.Fatal(err)
	}
	after, _ := e.Submit(context.Background(), order.ID, "100", "100")
	if after.Status != StateUnknown || f.calls != 1 {
		t.Fatalf("unknown expected %+v calls=%d", after, f.calls)
	}
	recovered, err := e.Reconcile(context.Background(), order.ID)
	if err != nil || recovered.Status != StateNew || recovered.OrderID != "901" {
		t.Fatalf("accepted but ACK lost %+v err=%v", recovered, err)
	}
	if _, err = e.Submit(context.Background(), order.ID, "100", "100"); err == nil || f.calls != 1 {
		t.Fatal("blind resubmission allowed")
	}
	e2, f2 := prepareS2(t)
	f2.createErr = errors.New("timeout")
	f2.queryErr = errors.New("-2013 unknown order")
	next, _ := e2.Prepare(context.Background(), requestS2("t2-05"))
	_, _ = e2.Submit(context.Background(), next.ID, "100", "100")
	unavailable, err := e2.Reconcile(context.Background(), next.ID)
	if err == nil || unavailable.Status != StateUnknown {
		t.Fatalf("query inconclusive %+v %v", unavailable, err)
	}
	if _, err = e2.Prepare(context.Background(), requestS2("t2-05-second")); err == nil {
		t.Fatal("unknown must keep pending slot")
	}
	if f2.calls != 1 {
		t.Fatal("new order submitted on uncertainty")
	}
}
func TestS2T06_RestartRecoversUsingSameClientID(t *testing.T) {
	e, f := prepareS2(t)
	order := mustPrepareSubmitS2(t, e, "t2-06")
	f.Lock()
	f.trades = []Trade{tradeS2(33, "3", "100", "300", "0", "BTC")}
	f.order.Status = "PARTIALLY_FILLED"
	f.order.ExecutedQty = "3"
	f.order.ExecutedQuoteQty = "300"
	f.Unlock()
	db, _ := orm.GetDB("default")
	if _, err := db.Exec("UPDATE spot_strategy_orders SET status='unknown',last_checked_at=0 WHERE id=?", order.ID); err != nil {
		t.Fatal(err)
	}
	restarted := NewMockEngine(db, f)
	n, err := restarted.RecoverPending(context.Background(), 10)
	if err != nil || n != 1 {
		t.Fatalf("restart recovered %d err=%v", n, err)
	}
	now, _ := restarted.Load(context.Background(), order.ID)
	if now.ClientOrderID != order.ClientOrderID || now.FilledQty != "3" || f.calls != 1 {
		t.Fatalf("restart identity changed: %+v", now)
	}
}
func TestS2T07_TransactionFailureNoInventedFills(t *testing.T) {
	e, f := prepareS2(t)
	order := mustPrepareSubmitS2(t, e, "t2-07")
	f.Lock()
	f.trades = []Trade{tradeS2(42, "10", "100", "1000", "0.01", "BTC")}
	f.order.Status = "FILLED"
	f.order.ExecutedQty = "10"
	f.order.ExecutedQuoteQty = "1000"
	f.Unlock()
	db, _ := orm.GetDB("default")
	if _, err := db.Exec(`CREATE TRIGGER s2_fail_fill BEFORE INSERT ON spot_strategy_fills
 BEGIN SELECT RAISE(ABORT,'simulated sqlite transaction failure'); END`); err != nil {
		t.Fatal(err)
	}
	defer db.Exec("DROP TRIGGER IF EXISTS s2_fail_fill")
	if _, err := e.Reconcile(context.Background(), order.ID); err == nil {
		t.Fatal("expected simulated DB failure")
	}
	if countTableS2(t, "spot_strategy_fills") != 0 {
		t.Fatal("phantom fills after rollback")
	}
	state, _ := e.Load(context.Background(), order.ID)
	if state.Status == StateFilled {
		t.Fatal("marked filled without commit")
	}
}
func TestS2T09_RateLimitAndMissingReadFailClosed(t *testing.T) {
	for _, apiErr := range []error{errors.New("429 Too many requests"), errors.New("418 banned"), errors.New("transport timeout")} {
		e, f := prepareS2(t)
		order := mustPrepareSubmitS2(t, e, "t2-09")
		f.queryErr = apiErr
		after, err := e.Reconcile(context.Background(), order.ID)
		if err == nil || after.Status != StateUnknown || !strings.HasPrefix(after.LastError, "spot_") {
			t.Fatalf("REST error not classified %s %+v %v", apiErr, after, err)
		}
		if _, err = e.Prepare(context.Background(), requestS2("t2-09-second")); err == nil {
			t.Fatal("new risk not blocked")
		}
	}
}
func TestS2T11_InvalidScopeAndExternalOrderRejected(t *testing.T) {
	e, f := prepareS2(t)
	req := requestS2("t2-11")
	req.AccountID = "lead"
	if _, err := e.Prepare(context.Background(), req); err == nil {
		t.Fatal("lead account accepted")
	}
	req = requestS2("t2-11")
	req.Side = "SELL"
	if _, err := e.Prepare(context.Background(), req); err == nil {
		t.Fatal("unowned sale accepted")
	}
	req = requestS2("t2-11")
	f.unmanaged = []OrderSnapshot{{OrderID: "777", Symbol: "BTCUSDT", Status: "NEW"}}
	if _, err := e.Prepare(context.Background(), req); err == nil {
		t.Fatal("unmanaged external order ignored")
	}
	if countTableS2(t, "spot_strategy_orders") != 0 {
		t.Fatal("invalid request created local order")
	}
}
func TestS2TradePaginationAndMismatch(t *testing.T) {
	e, f := prepareS2(t)
	order := mustPrepareSubmitS2(t, e, "t2-pages")
	f.Lock()
	for i := 1; i <= 505; i++ {
		f.trades = append(f.trades, tradeS2(int64(i), "0.01", "100", "1", "0", "BTC"))
	}
	f.order.Status = "PARTIALLY_FILLED"
	f.order.ExecutedQty = "5.05"
	f.order.ExecutedQuoteQty = "505"
	f.Unlock()
	got, err := e.Reconcile(context.Background(), order.ID)
	if err != nil || got.FilledQty != "5.05" || countTableS2(t, "spot_strategy_fills") != 505 {
		t.Fatalf("paginated fill %+v err=%v", got, err)
	}
	f.Lock()
	f.order.ExecutedQty = "6"
	f.Unlock()
	bad, err := e.Reconcile(context.Background(), order.ID)
	if !errors.Is(err, ErrReconciliation) || bad.Status != StateReconcile || bad.ReservedQty == "0" {
		t.Fatalf("mismatched cumulative filled did not block: %+v %v", bad, err)
	}
}
func TestS2LiveAdapterWriteDisabled(t *testing.T) {
	e, f := prepareS2(t)
	db, _ := orm.GetDB("default")
	live := NewReadOnlyEngine(db, f)
	if _, err := live.Prepare(context.Background(), requestS2("live")); !errors.Is(err, ErrLiveWriteDisabled) {
		t.Fatal(err)
	}
	if _, err := live.Submit(context.Background(), 1, "100", "100"); !errors.Is(err, ErrLiveWriteDisabled) {
		t.Fatal(err)
	}
	if _, err := live.CancelMock(context.Background(), 1); !errors.Is(err, ErrLiveWriteDisabled) {
		t.Fatal(err)
	}
	_ = e
}
func TestS2ExactDecimalAndFilterChecks(t *testing.T) {
	e, f := prepareS2(t)
	req := requestS2("bad-tick")
	req.Price = "100.5"
	if _, err := e.Prepare(context.Background(), req); err == nil {
		t.Fatal("invalid price tick allowed")
	}
	req = requestS2("bad-step")
	req.Quantity = "1.5"
	if _, err := e.Prepare(context.Background(), req); err == nil {
		t.Fatal("invalid quantity step allowed")
	}
	req = requestS2("bad-quote")
	f.instrument.QuoteAsset = "USDC"
	if _, err := e.Prepare(context.Background(), req); err == nil {
		t.Fatal("exchangeInfo quote mismatch ignored")
	}
	f.instrument.QuoteAsset = "USDT"
	f.free = "0"
	if _, err := e.Prepare(context.Background(), req); err == nil {
		t.Fatal("insufficient quote free ignored")
	}
}
func TestS2FeeAssetMissingBlocksRecord(t *testing.T) {
	e, f := prepareS2(t)
	order := mustPrepareSubmitS2(t, e, "t2-missing-fee")
	f.Lock()
	f.trades = []Trade{tradeS2(1, "1", "100", "100", "0.005", "")}
	f.order.Status = "PARTIALLY_FILLED"
	f.order.ExecutedQty = "1"
	f.order.ExecutedQuoteQty = "100"
	f.Unlock()
	if _, err := e.Reconcile(context.Background(), order.ID); err == nil {
		t.Fatal("fee asset missing should block verification")
	}
	if countTableS2(t, "spot_strategy_fills") != 0 {
		t.Fatal("unverified fee saved")
	}
}
func TestS2NoDuplicateIDsAfterTerminal(t *testing.T) {
	e, f := prepareS2(t)
	request := requestS2("signal-unique")
	request.ClientOrderID = "s510-user-specified"
	order, err := e.Prepare(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = e.Prepare(context.Background(), request); err == nil {
		t.Fatal("duplicate signal/client accepted")
	}
	_ = f
	if _, err = e.Submit(context.Background(), order.ID, "100", "100"); err != nil {
		t.Fatal(err)
	}
}
func TestS2SafeReadErrorDoesNotEchoSecrets(t *testing.T) {
	x := safeReadError(fmt.Errorf("BINANCE secret=DoNotLog 429"))
	if x != "spot_rest_rate_limited_429" {
		t.Fatalf("unsafe error text %s", x)
	}
}

// Verify reservation across different symbols using the SAME quote balance.
// A symbol-only pending-slot cannot prevent this type of overspend.
func TestS2SameQuoteBudgetAcrossSymbols(t *testing.T) {
	e, f := prepareS2(t)
	f.free = "1500"
	first, err := e.Prepare(context.Background(), requestS2("btc-pending"))
	if err != nil {
		t.Fatal(err)
	}
	if first.ReservedQty != "1000" {
		t.Fatalf("wrong quote reservation: %s", first.ReservedQty)
	}
	if _, err = orm.NewOrm().Insert(&models.SpotSymbols{Symbol: "ETHUSDT", Type: "USDT",
		StrategyTradeEnabled: 1, StrategyBudgetQuote: "2000", Technology: "{}", Strategy: "[]"}); err != nil {
		t.Fatal(err)
	}
	f.instrument.Symbol = "ETHUSDT"
	f.instrument.BaseAsset = "ETH"
	req := requestS2("eth-pending")
	req.Symbol = "ETHUSDT"
	if _, err = e.Prepare(context.Background(), req); err == nil {
		t.Fatal("cross-symbol quote overspend permitted")
	}
	if countTableS2(t, "spot_strategy_orders") != 1 {
		t.Fatal("overspend created second order")
	}
}

// A closed position's historical fill conflict must still freeze the symbol.
// Its pending_slot may already be NULL: a separate reconcile_required check
// is necessary before the next BUY.
func TestS2TamperedTerminalTradeBlocksNewRisk(t *testing.T) {
	e, f := prepareS2(t)
	order := mustPrepareSubmitS2(t, e, "closed-one")
	f.Lock()
	f.trades = []Trade{tradeS2(51, "10", "100", "1000", "0.01", "BNB")}
	f.order.Status = "FILLED"
	f.order.ExecutedQty = "10"
	f.order.ExecutedQuoteQty = "1000"
	f.Unlock()
	final, err := e.Reconcile(context.Background(), order.ID)
	if err != nil || final.Status != StateFilled {
		t.Fatalf("initial finished order %+v err=%v", final, err)
	}
	f.Lock()
	f.trades[0].Commission = "0.02"
	f.Unlock()
	if _, err = e.Reconcile(context.Background(), order.ID); err == nil {
		t.Fatal("conflicting tradeId silently accepted")
	}
	changed, err := e.Load(context.Background(), order.ID)
	if err != nil || changed.Status != StateReconcile {
		t.Fatalf("tampered historical fill not quarantined: %+v %v", changed, err)
	}
	if _, err = e.Prepare(context.Background(), requestS2("next-cycle")); err == nil {
		t.Fatal("unresolved terminal history permitted next BUY")
	}
}
func TestS2MarketZeroLotFilterFallsBackToNormalLot(t *testing.T) {
	e, f := prepareS2(t)
	f.instrument.MarketLotStep = "0.00000000"
	req := requestS2("zero-market-lot")
	req.Type = "MARKET"
	req.Price = ""
	req.EstimatePrice = "100"
	if _, err := e.Prepare(context.Background(), req); err != nil {
		t.Fatalf("disabled MARKET_LOT_SIZE cannot reject valid market mock: %v", err)
	}
}
func TestS2TradePageCapFailsClosed(t *testing.T) {
	e, f := prepareS2(t)
	order := mustPrepareSubmitS2(t, e, "pagelimit")
	f.Lock()
	for i := 1; i <= 10000; i++ {
		f.trades = append(f.trades, tradeS2(int64(i), "0.001", "100", "0.1", "0", "BTC"))
	}
	f.order.Status = "FILLED"
	f.order.ExecutedQty = "10"
	f.order.ExecutedQuoteQty = "1000"
	f.Unlock()
	after, err := e.Reconcile(context.Background(), order.ID)
	if !errors.Is(err, ErrReconciliation) || after.Status != StateReconcile {
		t.Fatalf("incomplete max pages failed to stop: %+v %v", after, err)
	}
	if countTableS2(t, "spot_strategy_fills") != 0 {
		t.Fatal("truncated pagination inserted partial fills")
	}
}
