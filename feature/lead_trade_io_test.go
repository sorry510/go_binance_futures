package feature

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/adshao/go-binance/v2/futures"
	binance "go_binance_futures/feature/api/binance"
	"go_binance_futures/models"
	"go_binance_futures/service/futuresownership"
	"go_binance_futures/types"
)

func TestLead03RejectMainAccountAtPrivateIOBoundary(t *testing.T) {
	main, err := binance.NewAccountClient(binance.MainAccountID, futures.NewClient("main", "secret"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := newLeadTradeIO(main); !errors.Is(err, errLeadTradeIOUnverified) {
		t.Fatalf("accepted Main credentials at Lead IO boundary: %v", err)
	}
	if _, err := newLeadTradeIO(nil); !errors.Is(err, errLeadTradeIOUnverified) {
		t.Fatalf("nil account accepted: %v", err)
	}
}

func TestLead03FuturesHedgePositionsFailClosed(t *testing.T) {
	long := &futures.PositionRisk{
		Symbol: "BTCUSDT", PositionSide: "LONG", PositionAmt: "1.5",
		Leverage: "4", MarginType: "cross", IsolatedWallet: "0",
		EntryPrice: "90", MarkPrice: "100", UnRealizedProfit: "15",
	}
	short := &futures.PositionRisk{
		Symbol: "ETHUSDT", PositionSide: "SHORT", PositionAmt: "-2.5",
		Leverage: "4", MarginType: "cross", IsolatedWallet: "0",
		EntryPrice: "100", MarkPrice: "90", UnRealizedProfit: "25",
	}
	rows, err := convertLeadAccountPositions([]*futures.PositionRisk{long, short})
	if err != nil || len(rows) != 2 || rows[0].SourceType != "api" ||
		rows[0].Amount != "1.5" || rows[1].Side != "SHORT" || rows[1].Amount != "-2.5" {
		t.Fatalf("Lead account transformation: %+v %v", rows, err)
	}
	for _, invalid := range []*futures.PositionRisk{
		nil,
		{Symbol: "BTCUSDT", PositionSide: "BOTH", PositionAmt: "1", Leverage: "4", EntryPrice: "1", MarkPrice: "1", UnRealizedProfit: "0"},
		{Symbol: "BTCUSDT", PositionSide: "LONG", PositionAmt: "-1", Leverage: "4", EntryPrice: "1", MarkPrice: "1", UnRealizedProfit: "0"},
		{Symbol: "BTCUSDT", PositionSide: "SHORT", PositionAmt: "1", Leverage: "4", EntryPrice: "1", MarkPrice: "1", UnRealizedProfit: "0"},
		{Symbol: "BTCUSDT", PositionSide: "LONG", PositionAmt: "NaN", Leverage: "4", EntryPrice: "1", MarkPrice: "1", UnRealizedProfit: "0"},
		{Symbol: "BTCUSDT", PositionSide: "LONG", PositionAmt: "1", Leverage: "0", EntryPrice: "1", MarkPrice: "1", UnRealizedProfit: "0"},
		{Symbol: "BTCUSDT", PositionSide: "LONG", PositionAmt: "1", Leverage: "4", EntryPrice: "1", MarkPrice: "NaN", UnRealizedProfit: "0"},
		{Symbol: "BTCUSDT", PositionSide: "LONG", PositionAmt: "1", Leverage: "4", EntryPrice: "1", MarkPrice: "0", UnRealizedProfit: "0"},
		{Symbol: "BTCUSDT", PositionSide: "LONG", PositionAmt: "1", Leverage: "4", EntryPrice: "0", MarkPrice: "1", UnRealizedProfit: "0"},
	} {
		if _, err := convertLeadAccountPositions([]*futures.PositionRisk{invalid}); !errors.Is(err, errLeadTradeIOUnverified) {
			t.Fatalf("unsafe Lead position accepted: %+v err=%v", invalid, err)
		}
	}
	if _, err := convertLeadAccountPositions([]*futures.PositionRisk{long, long}); !errors.Is(err, errLeadTradeIOUnverified) {
		t.Fatal("duplicate long legs accepted")
	}
}

func TestLead03OpenOrdersRemainReadOnlyAndAccountScoped(t *testing.T) {
	order := &futures.Order{OrderID: 123, Symbol: "BTCUSDT", ClientOrderID: "lead_o1",
		Side: futures.SideTypeBuy, PositionSide: futures.PositionSideTypeLong,
		Type: futures.OrderTypeLimit, Status: futures.OrderStatusTypeNew,
		OrigQuantity: "1", ExecutedQuantity: "0", Price: "99"}
	rows, err := convertLeadAccountOrders([]*futures.Order{order})
	if err != nil || len(rows) != 1 || rows[0].ClientOrderId != "lead_o1" || rows[0].OrderId != 123 {
		t.Fatalf("Lead open orders: %+v %v", rows, err)
	}
	for _, invalid := range []*futures.Order{
		nil,
		{OrderID: 3, Symbol: "BTCUSDT", PositionSide: futures.PositionSideTypeBoth, Status: futures.OrderStatusTypeNew},
		{OrderID: 3, Symbol: "BTCUSDT", PositionSide: futures.PositionSideTypeLong, Status: futures.OrderStatusTypeFilled},
	} {
		if _, err := convertLeadAccountOrders([]*futures.Order{invalid}); !errors.Is(err, errLeadTradeIOUnverified) {
			t.Fatalf("invalid open order accepted: %+v %v", invalid, err)
		}
	}
	if _, err := convertLeadAccountOrders([]*futures.Order{order, order}); !errors.Is(err, errLeadTradeIOUnverified) {
		t.Fatal("duplicate order accepted")
	}
	for _, invalidQty := range []string{"NaN", "0", "-1"} {
		bad := *order
		bad.OrigQuantity = invalidQty
		if _, err := convertLeadAccountOrders([]*futures.Order{&bad}); !errors.Is(err, errLeadTradeIOUnverified) {
			t.Fatalf("invalid order quantity %q accepted", invalidQty)
		}
	}
	bad := *order
	bad.ExecutedQuantity = "2"
	if _, err := convertLeadAccountOrders([]*futures.Order{&bad}); !errors.Is(err, errLeadTradeIOUnverified) {
		t.Fatal("overfilled exchange open order accepted")
	}
}

func TestLead03ManagedOwnershipCannotClaimManualQuantity(t *testing.T) {
	live := []types.FuturesPosition{
		{Symbol: "BTCUSDT", Side: "LONG", Amount: "1.5", Leverage: 4, EntryPrice: "90", MarkPrice: "100", UnrealizedProfit: "15"},
		{Symbol: "ETHUSDT", Side: "SHORT", Amount: "-2", Leverage: 4, EntryPrice: "100", MarkPrice: "90", UnrealizedProfit: "20"},
	}
	managed := []models.FuturesManagedPosition{
		{AccountID: "lead", Owner: futuresownership.OwnerAutoStrategy, Symbol: "BTCUSDT", PositionSide: "LONG", Status: futuresownership.PositionActive, ManagedQty: 1, EntryPrice: 90},
		{AccountID: "lead", Owner: futuresownership.OwnerAutoStrategy, Symbol: "ETHUSDT", PositionSide: "SHORT", Status: futuresownership.PositionActive, ManagedQty: 1, EntryPrice: 100},
	}
	got, err := verifyLeadManagedSnapshot(live, managed, nil)
	if err != nil || len(got) != 2 || got[0].Position.Amount != "1" || got[1].Position.Amount != "-1" {
		t.Fatalf("manual additions were included in managed amount: %+v %v", got, err)
	}
	managed[0].ManagedQty = 2 // manual reduction or stale DB
	if _, err := verifyLeadManagedSnapshot(live, managed, nil); !errors.Is(err, errLeadTradeIOUnverified) {
		t.Fatalf("over-owned quantity passed: %v", err)
	}
	managed[0].ManagedQty = 1
	managed[0].AccountID = "main"
	if _, err := verifyLeadManagedSnapshot(live, managed, nil); !errors.Is(err, errLeadTradeIOUnverified) {
		t.Fatalf("cross-account owned quantity passed: %v", err)
	}
	managed[0].AccountID = "lead"
	managed[0].Status = futuresownership.PositionReconcileRequired
	if _, err := verifyLeadManagedSnapshot(live, managed, nil); !errors.Is(err, errLeadTradeIOUnverified) {
		t.Fatal("reconciliation-required position was allowed to exit")
	}
	managed[0].Status = futuresownership.PositionActive
	managed[0].EntryPrice = 0
	if _, err := verifyLeadManagedSnapshot(live, managed, nil); !errors.Is(err, errLeadTradeIOUnverified) {
		t.Fatal("managed position without Fill entry cost was accepted")
	}
	managed[0].EntryPrice = 90
	if _, err := verifyLeadManagedSnapshot(live, managed,
		[]models.FuturesManagedOrder{{AccountID: "lead", Symbol: "BTCUSDT", Status: futuresownership.OrderReconcile}}); !errors.Is(err, errLeadTradeIOUnverified) {
		t.Fatalf("unsettled order was ignored: %v", err)
	}
	if _, err := verifyLeadManagedSnapshot(live, managed, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := verifyLeadManagedSnapshot(live, append(managed, managed[0]), nil); !errors.Is(err, errLeadTradeIOUnverified) {
		t.Fatal("duplicate managed position accepted")
	}
	if _, err := (new(leadTradeIO)).readPositions(context.Background()); !errors.Is(err, errLeadTradeIOUnverified) {
		t.Fatalf("invalid adapter read succeeded: %v", err)
	}
}

func TestLead03AccountReadsUseLeadSignedHTTPOnly(t *testing.T) {
	var positionCalls, orderCalls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet || req.Header.Get("X-MBX-APIKEY") != "lead-test-key" ||
			req.URL.Query().Get("signature") == "" {
			t.Errorf("unverified or non-Lead account read: method=%s path=%s", req.Method, req.URL.Path)
			w.WriteHeader(http.StatusForbidden)
			return
		}
		switch req.URL.Path {
		case "/fapi/v2/positionRisk":
			positionCalls.Add(1)
			_, _ = w.Write([]byte(`[{"symbol":"BTCUSDT","positionSide":"LONG","positionAmt":"1","leverage":"4","entryPrice":"90","markPrice":"100","unRealizedProfit":"10","marginType":"cross"}]`))
		case "/fapi/v1/openOrders":
			orderCalls.Add(1)
			_, _ = w.Write([]byte(`[{"symbol":"BTCUSDT","orderId":12,"clientOrderId":"lead_12","side":"BUY","positionSide":"LONG","type":"LIMIT","status":"NEW","price":"95","origQty":"1","executedQty":"0"}]`))
		default:
			t.Errorf("unexpected Lead REST endpoint: %s", req.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	sdk := futures.NewClient("lead-test-key", "lead-test-secret")
	sdk.SetApiEndpoint(server.URL)
	sdk.HTTPClient = server.Client()
	account, err := binance.NewAccountClient(binance.LeadAccountID, sdk)
	if err != nil {
		t.Fatal(err)
	}
	source, err := newLeadTradeIO(account)
	if err != nil {
		t.Fatal(err)
	}
	shared := &accountTradeRunner{AccountID: binance.LeadAccountID}
	if err := source.bindReadHooks(shared); err != nil ||
		shared.ReadPositions == nil || shared.ReadOpenOrders == nil || shared.SyncPositions == nil {
		t.Fatalf("Lead read boundary did not bind to shared strategy runner: %v", err)
	}
	positions, err := shared.ReadPositions(context.Background())
	if err != nil || len(positions) != 1 || positions[0].Amount != "1" || positions[0].Symbol != "BTCUSDT" {
		t.Fatalf("Lead position signed read failed: %+v %v", positions, err)
	}
	orders, err := shared.ReadOpenOrders(context.Background())
	if err != nil || len(orders) != 1 || orders[0].ClientOrderId != "lead_12" {
		t.Fatalf("Lead order signed read failed: %+v %v", orders, err)
	}
	if positionCalls.Load() != 1 || orderCalls.Load() != 1 {
		t.Fatalf("unexpected REST polling: positions=%d orders=%d", positionCalls.Load(), orderCalls.Load())
	}
}

func TestLead03BindReadHooksNeverUsesMainScope(t *testing.T) {
	r := &accountTradeRunner{AccountID: binance.LeadAccountID}
	if err := (new(leadTradeIO)).bindReadHooks(r); !errors.Is(err, errLeadTradeIOUnverified) {
		t.Fatalf("nil lead IO accepted: %v", err)
	}
	if r.ReadPositions != nil || r.ReadOpenOrders != nil || r.SyncPositions != nil {
		t.Fatal("unverified hooks mutated the runner")
	}
	main, err := binance.NewAccountClient(binance.MainAccountID, futures.NewClient("main", "secret"))
	if err != nil {
		t.Fatal(err)
	}
	forged := &leadTradeIO{account: main, ownership: futuresownership.Service{AccountID: "lead"}}
	if err := forged.bindReadHooks(r); !errors.Is(err, errLeadTradeIOUnverified) {
		t.Fatalf("Main client bound into Lead runner: %v", err)
	}
}
