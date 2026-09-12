package futuresownership

import (
	"context"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"go_binance_futures/binanceproxy"

	"github.com/adshao/go-binance/v2/futures"
	"github.com/beego/beego/v2/core/config"
)

// TestBinanceFuturesTestnetE2E is intentionally opt-in. It NEVER targets mainnet:
// futures.UseTestnet is forced before the client is created and credentials are
// accepted only from BINANCE_TESTNET_API_KEY / BINANCE_TESTNET_API_SECRET.
//
// Run with:
//
//	BINANCE_TESTNET_E2E=1 BINANCE_TESTNET_API_KEY=... BINANCE_TESTNET_API_SECRET=... \
//	  go test ./service/futuresownership -run TestBinanceFuturesTestnetE2E -v -count=1
func TestBinanceFuturesTestnetE2E(t *testing.T) {
	if os.Getenv("BINANCE_TESTNET_E2E") != "1" {
		t.Skip("set BINANCE_TESTNET_E2E=1 to run real Binance Futures Testnet orders")
	}
	key := strings.TrimSpace(os.Getenv("BINANCE_TESTNET_API_KEY"))
	secret := strings.TrimSpace(os.Getenv("BINANCE_TESTNET_API_SECRET"))
	if key == "" {
		key = testnetLocalConfigValue("binance::testnet_api_key")
	}
	if secret == "" {
		secret = testnetLocalConfigValue("binance::testnet_api_secret")
	}
	if key == "" || secret == "" {
		t.Fatal("Binance Futures Testnet credentials are required via BINANCE_TESTNET_API_KEY/BINANCE_TESTNET_API_SECRET or [binance] testnet_api_key/testnet_api_secret")
	}

	futures.UseTestnet = true
	defer func() { futures.UseTestnet = false }()
	client := futures.NewClient(key, secret)
	proxyURL := strings.TrimSpace(os.Getenv("BINANCE_TESTNET_PROXY_URL"))
	if proxyURL == "" {
		proxyURL = testnetLocalConfigValue("binance::proxy_url")
	}
	if proxyURL != "" {
		pool, err := binanceproxy.New(proxyURL)
		if err != nil {
			t.Fatalf("create testnet proxy: %v", err)
		}
		if pool.Enabled() {
			client.HTTPClient = pool.HTTPClient()
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	if _, err := client.NewGetAccountService().Do(ctx); err != nil {
		t.Fatalf("Binance Futures Testnet authentication failed: %v", err)
	}
	mode, err := client.NewGetPositionModeService().Do(ctx)
	if err != nil {
		t.Fatalf("read testnet position mode: %v", err)
	}
	if !mode.DualSidePosition {
		if os.Getenv("BINANCE_TESTNET_ALLOW_HEDGE_MODE_CHANGE") != "1" {
			t.Fatal("testnet account is not in Hedge Mode; enable it or set BINANCE_TESTNET_ALLOW_HEDGE_MODE_CHANGE=1 on a dedicated testnet account")
		}
		if err := client.NewChangePositionModeService().DualSide(true).Do(ctx); err != nil {
			t.Fatalf("enable testnet Hedge Mode: %v", err)
		}
	}

	symbol := strings.ToUpper(strings.TrimSpace(os.Getenv("BINANCE_TESTNET_SYMBOL")))
	if symbol == "" {
		symbol = "BTCUSDT"
	}
	preflightEmptySymbol(t, ctx, client, symbol)
	defer cleanupTestnetSymbol(t, client, symbol)

	meta := loadTestnetSymbolMeta(t, ctx, client, symbol)
	if _, err := client.NewChangeLeverageService().Symbol(symbol).Leverage(10).Do(ctx); err != nil {
		t.Fatalf("set testnet leverage: %v", err)
	}

	prepareOwnershipDB(t)
	ownership := testService()
	broker := &testnetOrderBroker{client: client}
	executor := Executor{Ownership: ownership, Broker: broker}
	account := &testnetAccountSource{client: client}
	reconciler := Reconciler{Ownership: ownership, Executor: executor, Account: account}
	prefix := fmt.Sprintf("v35e2e_%d", time.Now().UnixMilli()%1_000_000_000)

	runScenario := func(t *testing.T, fn func()) {
		t.Helper()
		prepareOwnershipDB(t)
		preflightEmptySymbol(t, ctx, client, symbol)
		defer cleanupTestnetSymbol(t, client, symbol)
		fn()
	}
	t.Run("open long then close long", func(t *testing.T) {
		runScenario(t, func() { runOpenCloseE2E(t, ctx, executor, client, symbol, "LONG", meta.unitQty, prefix+"_l") })
	})
	t.Run("open short then close short", func(t *testing.T) {
		runScenario(t, func() { runOpenCloseE2E(t, ctx, executor, client, symbol, "SHORT", meta.unitQty, prefix+"_s") })
	})
	t.Run("take profit and stop loss protection", func(t *testing.T) {
		runScenario(t, func() { runProtectionE2E(t, ctx, ownership, executor, client, symbol, meta, prefix+"_p") })
	})
	t.Run("manual partial reduce then close managed remainder", func(t *testing.T) {
		runScenario(t, func() { runManualReduceE2E(t, ctx, ownership, executor, reconciler, client, symbol, meta, prefix+"_m") })
	})
	preflightEmptySymbol(t, ctx, client, symbol)
	t.Logf("testnet final state clean: %s has no position, normal order, or algo order", symbol)
}

func testnetLocalConfigValue(key string) string {
	cfg, err := config.NewConfig("ini", "../../conf/app.conf")
	if err != nil {
		return ""
	}
	value, err := cfg.String(key)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(value)
}

type testnetSymbolMeta struct {
	unitQty   float64
	stepSize  float64
	tickSize  float64
	lastPrice float64
}

func loadTestnetSymbolMeta(t *testing.T, ctx context.Context, client *futures.Client, symbol string) testnetSymbolMeta {
	t.Helper()
	prices, err := client.NewListPricesService().Symbol(symbol).Do(ctx)
	if err != nil || len(prices) == 0 {
		t.Fatalf("load %s testnet price: %v", symbol, err)
	}
	lastPrice, err := strconv.ParseFloat(prices[0].Price, 64)
	if err != nil || lastPrice <= 0 {
		t.Fatalf("invalid %s testnet price %q", symbol, prices[0].Price)
	}
	info, err := client.NewExchangeInfoService().Do(ctx)
	if err != nil {
		t.Fatalf("load testnet exchange info: %v", err)
	}
	for _, item := range info.Symbols {
		if item.Symbol != symbol {
			continue
		}
		lot := item.MarketLotSizeFilter()
		if lot == nil {
			t.Fatal("testnet symbol has no MARKET_LOT_SIZE filter")
		}
		step, _ := strconv.ParseFloat(lot.StepSize, 64)
		minQty, _ := strconv.ParseFloat(lot.MinQuantity, 64)
		minNotional := 0.0
		if f := item.MinNotionalFilter(); f != nil {
			minNotional, _ = strconv.ParseFloat(f.Notional, 64)
		}
		priceFilter := item.PriceFilter()
		if priceFilter == nil {
			t.Fatal("testnet symbol has no PRICE_FILTER")
		}
		tick, _ := strconv.ParseFloat(priceFilter.TickSize, 64)
		if step <= 0 || tick <= 0 {
			t.Fatalf("invalid testnet symbol filters: step=%v tick=%v", step, tick)
		}
		unit := math.Max(minQty, (minNotional/lastPrice)*1.10)
		unit = ceilToStep(unit, step)
		return testnetSymbolMeta{unitQty: unit, stepSize: step, tickSize: tick, lastPrice: lastPrice}
	}
	t.Fatalf("testnet symbol %s not found", symbol)
	return testnetSymbolMeta{}
}

func ceilToStep(value, step float64) float64 {
	return math.Ceil((value-1e-12)/step) * step
}

func roundToStep(value, step float64) float64 {
	if step <= 0 {
		return value
	}
	return math.Round(value/step) * step
}

func formatTestnetDecimal(value float64) string {
	text := strconv.FormatFloat(value, 'f', 8, 64)
	text = strings.TrimRight(strings.TrimRight(text, "0"), ".")
	if text == "" || text == "-0" {
		return "0"
	}
	return text
}

func formatTestnetQty(value, step float64) string {
	value = roundToStep(value, step)
	precision := 0
	stepText := strconv.FormatFloat(step, 'f', -1, 64)
	if dot := strings.IndexByte(stepText, '.'); dot >= 0 {
		precision = len(strings.TrimRight(stepText[dot+1:], "0"))
	}
	return strconv.FormatFloat(value, 'f', precision, 64)
}

func preflightEmptySymbol(t *testing.T, ctx context.Context, client *futures.Client, symbol string) {
	t.Helper()
	orders, err := client.NewListOpenOrdersService().Symbol(symbol).Do(ctx)
	if err != nil {
		t.Fatalf("list testnet open orders: %v", err)
	}
	if len(orders) != 0 {
		t.Fatalf("refuse E2E: %s already has %d open testnet orders", symbol, len(orders))
	}
	algoOrders, err := client.NewListOpenAlgoOrdersService().Symbol(symbol).Do(ctx)
	if err != nil {
		t.Fatalf("list testnet open algo orders: %v", err)
	}
	if len(algoOrders) != 0 {
		t.Fatalf("refuse E2E: %s already has %d open testnet algo orders", symbol, len(algoOrders))
	}
	positions, err := client.NewGetPositionRiskService().Symbol(symbol).Do(ctx)
	if err != nil {
		t.Fatalf("load testnet positions: %v", err)
	}
	for _, position := range positions {
		qty, _ := strconv.ParseFloat(position.PositionAmt, 64)
		if math.Abs(qty) > 1e-12 {
			t.Fatalf("refuse E2E: %s %s already has testnet quantity %s", symbol, position.PositionSide, position.PositionAmt)
		}
	}
}

func runOpenCloseE2E(t *testing.T, ctx context.Context, executor Executor, client *futures.Client, symbol, positionSide string, qty float64, prefix string) {
	t.Helper()
	openSide, closeSide := "BUY", "SELL"
	if positionSide == "SHORT" {
		openSide, closeSide = "SELL", "BUY"
	}
	openID := prefix + "_o"
	if _, err := executor.Execute(ctx, OrderRequest{Owner: OwnerAutoStrategy, Symbol: symbol, PositionSide: positionSide, Intent: IntentOpen, Side: openSide, OrderType: "MARKET", Quantity: qty, SourceRef: prefix, ClientOrderID: openID}); err != nil {
		t.Fatalf("open %s: %v", positionSide, err)
	}
	waitTestnetQty(t, ctx, client, symbol, positionSide, qty)
	closeID := prefix + "_c"
	if _, err := executor.Execute(ctx, OrderRequest{Owner: OwnerAutoStrategy, Symbol: symbol, PositionSide: positionSide, Intent: IntentClose, Side: closeSide, OrderType: "MARKET", Quantity: qty, SourceRef: prefix, ClientOrderID: closeID}); err != nil {
		t.Fatalf("close %s: %v", positionSide, err)
	}
	waitTestnetQty(t, ctx, client, symbol, positionSide, 0)
}

func runProtectionE2E(t *testing.T, ctx context.Context, ownership Service, executor Executor, client *futures.Client, symbol string, meta testnetSymbolMeta, prefix string) {
	t.Helper()
	qty := meta.unitQty
	if _, err := executor.Execute(ctx, OrderRequest{Owner: OwnerAutoStrategy, Symbol: symbol, PositionSide: "LONG", Intent: IntentOpen, Side: "BUY", OrderType: "MARKET", Quantity: qty, SourceRef: prefix, ClientOrderID: prefix + "_o"}); err != nil {
		t.Fatalf("open protection LONG: %v", err)
	}
	waitTestnetQty(t, ctx, client, symbol, "LONG", qty)
	prices, err := client.NewListPricesService().Symbol(symbol).Do(ctx)
	if err != nil || len(prices) == 0 {
		t.Fatalf("refresh protection price: %v", err)
	}
	price, _ := strconv.ParseFloat(prices[0].Price, 64)
	tpPrice := math.Ceil((price*1.05)/meta.tickSize) * meta.tickSize
	slPrice := math.Floor((price*0.95)/meta.tickSize) * meta.tickSize

	if _, err := executor.Execute(ctx, OrderRequest{Owner: OwnerAutoStrategy, Symbol: symbol, PositionSide: "LONG", Intent: IntentTakeProfit, Side: "SELL", OrderType: "TAKE_PROFIT_MARKET", Quantity: qty, StopPrice: tpPrice, SourceRef: prefix, ClientOrderID: prefix + "_tp"}); err != nil {
		t.Fatalf("create testnet TP: %v", err)
	}
	if _, err := executor.Execute(ctx, OrderRequest{Owner: OwnerAutoStrategy, Symbol: symbol, PositionSide: "LONG", Intent: IntentStopLoss, Side: "SELL", OrderType: "STOP_MARKET", Quantity: qty, StopPrice: slPrice, SourceRef: prefix, ClientOrderID: prefix + "_sl"}); err != nil {
		t.Fatalf("create testnet SL: %v", err)
	}
	for _, clientID := range []string{prefix + "_tp", prefix + "_sl"} {
		row, err := findTestOrder(clientID)
		if err != nil {
			t.Fatalf("load managed protection %s: %v", clientID, err)
		}
		if row.ExchangeOrderID == "" || !liveOrderStatus(row.Status) {
			t.Fatalf("protection %s not live: %+v", clientID, row)
		}
		if err := executor.Cancel(ctx, OwnerAutoStrategy, row); err != nil {
			t.Fatalf("cancel protection %s: %v", clientID, err)
		}
	}
	position, err := ownership.GetPosition(ctx, OwnerAutoStrategy, symbol, "LONG")
	if err != nil {
		t.Fatalf("load managed protection position: %v", err)
	}
	if _, err := executor.Execute(ctx, OrderRequest{Owner: OwnerAutoStrategy, Symbol: symbol, PositionSide: "LONG", Intent: IntentClose, Side: "SELL", OrderType: "MARKET", Quantity: position.ManagedQty, SourceRef: prefix, ClientOrderID: prefix + "_c"}); err != nil {
		t.Fatalf("close protection LONG: %v", err)
	}
	waitTestnetQty(t, ctx, client, symbol, "LONG", 0)
}

func runManualReduceE2E(t *testing.T, ctx context.Context, ownership Service, executor Executor, reconciler Reconciler, client *futures.Client, symbol string, meta testnetSymbolMeta, prefix string) {
	t.Helper()
	total := roundToStep(meta.unitQty*3, meta.stepSize)
	t.Logf("manual-reduce meta unitQty=%0.18f step=%0.18f total=%0.18f submit=%q", meta.unitQty, meta.stepSize, total, formatTestnetDecimal(total))
	if _, err := executor.Execute(ctx, OrderRequest{Owner: OwnerAutoStrategy, Symbol: symbol, PositionSide: "LONG", Intent: IntentOpen, Side: "BUY", OrderType: "MARKET", Quantity: total, SourceRef: prefix, ClientOrderID: prefix + "_o"}); err != nil {
		t.Fatalf("open manual-reduce LONG: %v", err)
	}
	waitTestnetQty(t, ctx, client, symbol, "LONG", total)

	manualID := prefix + "_manual"
	_, err := client.NewCreateOrderService().Symbol(symbol).Side(futures.SideTypeSell).PositionSide(futures.PositionSideTypeLong).Type(futures.OrderTypeMarket).Quantity(formatTestnetQty(meta.unitQty, meta.stepSize)).NewClientOrderID(manualID).Do(ctx)
	if err != nil {
		t.Fatalf("manual testnet partial reduce: %v", err)
	}
	remaining := roundToStep(total-meta.unitQty, meta.stepSize)
	waitTestnetQty(t, ctx, client, symbol, "LONG", remaining)
	if _, err := reconciler.ReconcileOwner(ctx, OwnerAutoStrategy); err != nil {
		t.Fatalf("reconcile after manual reduce: %v", err)
	}
	managed, err := ownership.GetPosition(ctx, OwnerAutoStrategy, symbol, "LONG")
	if err != nil {
		t.Fatalf("load managed qty after manual reduce: %v", err)
	}
	if math.Abs(managed.ManagedQty-remaining) > math.Max(1e-10, meta.unitQty/1000) {
		t.Fatalf("managed qty did not shrink: got %.12f want %.12f", managed.ManagedQty, remaining)
	}
	if _, err := executor.Execute(ctx, OrderRequest{Owner: OwnerAutoStrategy, Symbol: symbol, PositionSide: "LONG", Intent: IntentClose, Side: "SELL", OrderType: "MARKET", Quantity: managed.ManagedQty, SourceRef: prefix, ClientOrderID: prefix + "_c"}); err != nil {
		t.Fatalf("close reconciled remainder: %v", err)
	}
	waitTestnetQty(t, ctx, client, symbol, "LONG", 0)
}

func waitTestnetQty(t *testing.T, ctx context.Context, client *futures.Client, symbol, positionSide string, want float64) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for {
		positions, err := client.NewGetPositionRiskService().Symbol(symbol).Do(ctx)
		if err == nil {
			for _, p := range positions {
				if !strings.EqualFold(p.PositionSide, positionSide) {
					continue
				}
				qty, _ := strconv.ParseFloat(p.PositionAmt, 64)
				if math.Abs(math.Abs(qty)-want) <= math.Max(1e-10, want/10000) {
					return
				}
			}
		}
		if time.Now().After(deadline) {
			t.Fatalf("testnet %s %s quantity did not reach %.12f", symbol, positionSide, want)
		}
		select {
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		case <-time.After(200 * time.Millisecond):
		}
	}
}

type testnetOrderBroker struct{ client *futures.Client }

func (b *testnetOrderBroker) Submit(ctx context.Context, request OrderRequest, clientOrderID string) (ExchangeOrder, error) {
	side := futures.SideTypeBuy
	if strings.EqualFold(request.Side, "SELL") {
		side = futures.SideTypeSell
	}
	positionSide := futures.PositionSideTypeLong
	if strings.EqualFold(request.PositionSide, "SHORT") {
		positionSide = futures.PositionSideTypeShort
	}
	orderType := strings.ToUpper(strings.TrimSpace(request.OrderType))
	if isAlgoManagedOrderType(orderType) {
		service := b.client.NewCreateAlgoOrderService().Symbol(request.Symbol).Side(side).PositionSide(positionSide).
			Type(futures.AlgoOrderType(orderType)).Quantity(formatTestnetDecimal(request.Quantity)).ClientAlgoId(clientOrderID)
		if request.StopPrice > 0 {
			service = service.TriggerPrice(formatTestnetDecimal(request.StopPrice))
		}
		algo, err := service.Do(ctx)
		if err != nil {
			return ExchangeOrder{}, err
		}
		return exchangeFromAlgoCreate(algo), nil
	}
	service := b.client.NewCreateOrderService().Symbol(request.Symbol).Side(side).PositionSide(positionSide).Type(futures.OrderType(orderType)).Quantity(formatTestnetDecimal(request.Quantity)).NewClientOrderID(clientOrderID)
	if futures.OrderType(orderType) == futures.OrderTypeLimit {
		service = service.TimeInForce(futures.TimeInForceTypeGTC).Price(formatTestnetDecimal(request.Price))
	}
	order, err := service.Do(ctx)
	if err != nil {
		return ExchangeOrder{}, err
	}
	return exchangeFromCreate(order), nil
}

func (b *testnetOrderBroker) Lookup(ctx context.Context, symbol, clientOrderID, orderType string) (ExchangeOrder, error) {
	if isAlgoManagedOrderType(orderType) {
		algo, err := b.client.NewGetAlgoOrderService().ClientAlgoID(clientOrderID).Do(ctx)
		if err != nil {
			return ExchangeOrder{}, err
		}
		result := ExchangeOrder{ExchangeOrderID: strconv.FormatInt(algo.AlgoId, 10), ClientOrderID: algo.ClientAlgoId, Status: string(algo.AlgoStatus)}
		if strings.TrimSpace(algo.ActualOrderId) == "" || strings.TrimSpace(algo.ActualOrderId) == "0" {
			return result, nil
		}
		actualID, err := strconv.ParseInt(strings.TrimSpace(algo.ActualOrderId), 10, 64)
		if err != nil {
			return result, err
		}
		actual, err := b.client.NewGetOrderService().Symbol(algo.Symbol).OrderID(actualID).Do(ctx)
		if err != nil {
			return result, err
		}
		actualResult := exchangeFromOrder(actual)
		result.Status, result.FilledQty, result.AveragePrice = actualResult.Status, actualResult.FilledQty, actualResult.AveragePrice
		return result, nil
	}
	order, err := b.client.NewGetOrderService().Symbol(symbol).OrigClientOrderID(clientOrderID).Do(ctx)
	if err != nil {
		return ExchangeOrder{}, err
	}
	return exchangeFromOrder(order), nil
}

func (b *testnetOrderBroker) Cancel(ctx context.Context, symbol string, orderID int64, orderType string) error {
	if isAlgoManagedOrderType(orderType) {
		_, err := b.client.NewCancelAlgoOrderService().AlgoID(orderID).Do(ctx)
		return err
	}
	_, err := b.client.NewCancelOrderService().Symbol(symbol).OrderID(orderID).Do(ctx)
	return err
}

type testnetAccountSource struct{ client *futures.Client }

func (s *testnetAccountSource) Positions(ctx context.Context) ([]AccountPosition, error) {
	rows, err := s.client.NewGetPositionRiskService().Do(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]AccountPosition, 0)
	for _, row := range rows {
		qty, err := strconv.ParseFloat(row.PositionAmt, 64)
		if err != nil || math.Abs(qty) <= 1e-12 {
			continue
		}
		out = append(out, AccountPosition{Symbol: row.Symbol, PositionSide: row.PositionSide, Quantity: qty})
	}
	return out, nil
}

func cleanupTestnetSymbol(t *testing.T, client *futures.Client, symbol string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	algoOrders, algoErr := client.NewListOpenAlgoOrdersService().Symbol(symbol).Do(ctx)
	if algoErr == nil {
		for _, order := range algoOrders {
			if !strings.HasPrefix(order.ClientAlgoId, "v35e2e_") {
				continue
			}
			_, _ = client.NewCancelAlgoOrderService().AlgoID(order.AlgoId).Do(ctx)
		}
	}
	orders, err := client.NewListOpenOrdersService().Symbol(symbol).Do(ctx)
	if err == nil {
		for _, order := range orders {
			if !strings.HasPrefix(order.ClientOrderID, "v35e2e_") {
				continue
			}
			_, _ = client.NewCancelOrderService().Symbol(symbol).OrderID(order.OrderID).Do(ctx)
		}
	}
	positions, err := client.NewGetPositionRiskService().Symbol(symbol).Do(ctx)
	if err != nil {
		t.Logf("cleanup position query failed: %v", err)
		return
	}
	for _, position := range positions {
		qty, _ := strconv.ParseFloat(position.PositionAmt, 64)
		qty = math.Abs(qty)
		if qty <= 1e-12 {
			continue
		}
		side := futures.SideTypeSell
		positionSide := futures.PositionSideTypeLong
		if strings.EqualFold(position.PositionSide, "SHORT") {
			side = futures.SideTypeBuy
			positionSide = futures.PositionSideTypeShort
		}
		_, closeErr := client.NewCreateOrderService().Symbol(symbol).Side(side).PositionSide(positionSide).Type(futures.OrderTypeMarket).Quantity(formatTestnetDecimal(qty)).NewClientOrderID(fmt.Sprintf("v35e2e_cleanup_%d", time.Now().UnixNano()%1_000_000_000)).Do(ctx)
		if closeErr != nil {
			t.Logf("cleanup %s %s qty %.12f failed: %v", symbol, position.PositionSide, qty, closeErr)
		}
	}
}
