package feature

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"go_binance_futures/feature/strategy/line"
	"go_binance_futures/models"
	"go_binance_futures/service/spotledger"

	"github.com/adshao/go-binance/v2/futures"
	"github.com/beego/beego/v2/client/orm"
)

// A real SpotRuleEvaluator, strict market-data validation, real Processor,
// real SQL Fill ledger and the 06-5 read model. No live Binance request.
func spot0674StrictFixture(t *testing.T) (*SpotQuantRunner, *spot0673Exchange, orm.Ormer, *bool, *bool) {
	t.Helper()
	r, ex, _, db := spot0673Fixture(t)
	spot0673Enable(t, db)
	technology := `{"ma":[{"name":"ma_5","enable":true,"kline_interval":"5m","period":5}]}`
	strategy := `[{"name":"strict Spot MA","type":"long","enable":true,
		"code":"NowPrice > 90 and ma_5.Data[0] > 0 and kline_5m.Close[0] > 0"},
		{"name":"close","type":"close_long","enable":true,"code":"ROI >= 8"}]`
	if _, err := db.Raw(`UPDATE spot_symbols SET technology=?,strategy=?,
		close='100',open='100',low='90',high='120',updateTime=?
		WHERE symbol='BTCUSDT'`, technology, strategy, time.Now().UnixMilli()).Exec(); err != nil {
		t.Fatal(err)
	}
	staleKline := new(bool)
	incompleteKline := new(bool)
	klineLoader := func(symbol, interval string, limit int) ([]*futures.Kline, error) {
		if symbol != "BTCUSDT" || (interval != "5m" && interval != "3d") || limit < 5 {
			return nil, fmt.Errorf("unexpected Spot indicator market request")
		}
		now := time.Now().UTC()
		span := 5 * time.Minute
		latest := now.Truncate(span)
		if interval == "3d" {
			// Valid FAKE 3d Spot bars for strict indicator testing;
			// NOT an authoritative Binance Kline OpenTime provider.
			span = 72 * time.Hour
			latest = now.Add(-24 * time.Hour)
		}
		if *staleKline {
			latest = latest.Add(-30 * time.Minute)
		}
		if *incompleteKline {
			limit = 3
		}
		rows := make([]*futures.Kline, limit)
		for i := range rows {
			open := latest.Add(time.Duration(i-limit+1) * span)
			rows[i] = &futures.Kline{
				Open: "100", High: "110", Low: "90", Close: "105",
				OpenTime:  open.UnixMilli(),
				CloseTime: open.Add(span).UnixMilli() - 1,
			}
		}
		return rows, nil
	}
	r.strategy = line.SpotRuleEvaluator{
		LoadEnvironment: func(coin models.SpotSymbols) (map[string]interface{}, error) {
			return line.LoadVerifiedSpotRuleEnvWithSource(coin, time.Now(), ex.price, klineLoader)
		},
		LoadKlines: klineLoader,
	}
	return r, ex, db, staleKline, incompleteKline
}

func TestStage0674StrictSpotMarketEnvironmentThroughRealProcessor(t *testing.T) {
	r, ex, db, stale, incomplete := spot0674StrictFixture(t)
	// The ticker age is checked even though the exchange Fake has a
	// valid live account and price: fresh WS cannot rescue old OHLC.
	if _, err := db.Raw("UPDATE spot_symbols SET updateTime=? WHERE symbol='BTCUSDT'",
		time.Now().Add(-8*time.Minute).UnixMilli()).Exec(); err != nil {
		t.Fatal(err)
	}
	if err := r.Run(context.Background()); err == nil || ex.places != 0 {
		t.Fatalf("stale Spot OHLC produced BUY: err=%v places=%d", err, ex.places)
	}
	if _, err := db.Raw("UPDATE spot_symbols SET updateTime=? WHERE symbol='BTCUSDT'",
		time.Now().UnixMilli()).Exec(); err != nil {
		t.Fatal(err)
	}
	*incomplete = true
	if err := r.Run(context.Background()); err == nil || ex.places != 0 {
		t.Fatalf("MA5 with three bars produced BUY: err=%v places=%d", err, ex.places)
	}
	*incomplete = false
	*stale = true
	if err := r.Run(context.Background()); err == nil || ex.places != 0 {
		t.Fatalf("30m-old 5m Kline produced BUY: err=%v places=%d", err, ex.places)
	}
	*stale = false
	if err := r.Run(context.Background()); err != nil {
		t.Fatalf("strict healthy Spot market BUY: %v", err)
	}
	if ex.places != 1 {
		t.Fatalf("real Processor did not BUY valid Spot indicator signal: places=%d", ex.places)
	}
	pos, exists, err := r.activeSpotPosition("BTCUSDT")
	if err != nil || !exists || pos.Status != "OPEN" {
		t.Fatalf("verified strategy inventory missing: %+v %v", pos, err)
	}
	// The strict strategy consumes the SAME fake live price as the
	// Processor; DB 24h OHLC remains valid but its last close stays 100.
	ex.price = "110"
	if _, err := db.Raw("UPDATE spot_symbols SET updateTime=? WHERE symbol='BTCUSDT'",
		time.Now().Add(-8*time.Minute).UnixMilli()).Exec(); err != nil {
		t.Fatal(err)
	}
	if err := r.Run(context.Background()); err == nil || ex.places != 1 {
		t.Fatalf("stale Spot ticker incorrectly closed verified inventory: err=%v places=%d", err, ex.places)
	}
	if _, err := db.Raw("UPDATE spot_symbols SET updateTime=? WHERE symbol='BTCUSDT'",
		time.Now().UnixMilli()).Exec(); err != nil {
		t.Fatal(err)
	}
	if err := r.Run(context.Background()); err != nil {
		t.Fatalf("strict Spot indicator SELL: %v", err)
	}
	if ex.places != 2 {
		t.Fatalf("strict Spot evaluator failed to SELL: %d", ex.places)
	}
	closed, err := r.ledger.Accounting(pos.ID, "110")
	if err != nil || closed.Position.Status != "CLOSED" {
		t.Fatalf("verified BUY->SELL ledger not CLOSED: %+v %v", closed, err)
	}
	for i := 0; i < 5; i++ {
		if err := r.Run(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	if ex.places != 2 {
		t.Fatalf("duplicate same Kline BUY after close: %d", ex.places)
	}
}

func TestStage0674ThreeDayCandleRequiresExchangeOpenTime(t *testing.T) {
	now := time.Date(2026, 11, 13, 12, 0, 0, 0, time.UTC)
	if !spotCandleOpen(now, "3d").IsZero() {
		t.Fatal("3d Unix epoch truncation must never be used as an authoritative exchange candle")
	}
	var observed []string
	for _, tc := range []struct {
		name string
		open time.Time
	}{
		{"SpotTestnet_public_2026_11_13", time.Date(2026, 11, 12, 0, 0, 0, 0, time.UTC)},
		{"SpotMainnet_public_2026_11_13", time.Date(2026, 11, 13, 0, 0, 0, 0, time.UTC)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r, ex, _, db := spot0673Fixture(t)
			spot0673Enable(t, db)
			if _, err := db.Raw("UPDATE spot_symbols SET technology=? WHERE symbol='BTCUSDT'",
				`{"ema":[{"name":"ema_3d","enable":true,"kline_interval":"3d","period":5}]}`).Exec(); err != nil {
				t.Fatal(err)
			}
			r.clock = func() time.Time { return now }
			if err := r.Run(context.Background()); !errors.Is(err, spotledger.ErrNeedsReview) || ex.places != 0 {
				t.Fatalf("3d with NO authoritative market source placed an order: err=%v places=%d", err, ex.places)
			}
			r.spot3dOpen = func(symbol string, at time.Time) (time.Time, error) {
				if symbol != "BTCUSDT" || !at.Equal(now) {
					return time.Time{}, fmt.Errorf("wrong market")
				}
				return tc.open, nil
			}
			if err := r.Run(context.Background()); err != nil {
				t.Fatalf("3d with source was blocked: %v", err)
			}
			if ex.places != 1 {
				t.Fatalf("3d source failed to establish event order: %d", ex.places)
			}
			orders, err := r.ledger.ListUnsettledOrders("BTCUSDT")
			if err != nil {
				t.Fatal(err)
			}
			if len(orders) != 0 {
				t.Fatalf("fully filled 3d BUY left unsettled: %v", orders)
			}
			pos, exists, err := r.activeSpotPosition("BTCUSDT")
			if err != nil || !exists {
				t.Fatalf("missing 3d inventory: %v", err)
			}
			local, err := r.ledger.ListOrders(pos.ID)
			if err != nil || len(local) != 1 {
				t.Fatalf("missing 3d BUY local order: %v", err)
			}
			interval, opened, ok := spotStoredBUYEvent(local[0].ClientOrderID)
			if !ok || interval != "3d" || !opened.Equal(tc.open) {
				t.Fatalf("3d clientID did not preserve actual exchange event open: %s %s %v", interval, opened, ok)
			}
			observed = append(observed, local[0].ClientOrderID)
			for i := 0; i < 5; i++ {
				if err := r.Run(context.Background()); err != nil {
					t.Fatal(err)
				}
			}
			if ex.places != 1 {
				t.Fatalf("3d same candle duplicated order: %d", ex.places)
			}
		})
	}
	if len(observed) != 2 || observed[0] == observed[1] {
		t.Fatal("distinct Mainnet/Testnet 3d opens produced the same client ID")
	}
}

func TestStage0674ThreeDayStaleSourceFailsClosed(t *testing.T) {
	r, ex, _, db := spot0673Fixture(t)
	spot0673Enable(t, db)
	if _, err := db.Raw("UPDATE spot_symbols SET technology=? WHERE symbol='BTCUSDT'",
		`{"ma":[{"name":"ma_3d","enable":true,"kline_interval":"3d","period":5}]}`).Exec(); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 11, 13, 12, 0, 0, 0, time.UTC)
	r.clock = func() time.Time { return now }
	r.spot3dOpen = func(_ string, _ time.Time) (time.Time, error) {
		return time.Date(2026, 11, 9, 0, 0, 0, 0, time.UTC), nil
	}
	if err := r.Run(context.Background()); !errors.Is(err, spotledger.ErrNeedsReview) || ex.places != 0 {
		t.Fatalf("stale wrong 3d Kline must fail-closed: err=%v places=%d", err, ex.places)
	}
}

// Review 0674 F1: a validated OPEN Spot position must not get stuck merely
// because the user switched the indicator's timeframe to 3d after BUY.
// Any valid Spot EXIT signal is bound to Position ID, never 3d BUY timing.
func TestReview0674ThreeDayMissingSourceAllowsVerifiedPositionSELL(t *testing.T) {
	r, ex, _, db := spot0673Fixture(t)
	spot0673Enable(t, db)
	spot0673Tick(t, r)
	if ex.places != 1 {
		t.Fatalf("initial verified BUY count=%d", ex.places)
	}
	pos, exists, err := r.activeSpotPosition("BTCUSDT")
	if err != nil || !exists || pos.Status != "OPEN" {
		t.Fatalf("missing verified inventory: %+v %v", pos, err)
	}
	technology := `{"ema":[{"name":"ema_3d","enable":true,"kline_interval":"3d","period":5}]}`
	if _, err := db.Raw("UPDATE spot_symbols SET technology=? WHERE symbol='BTCUSDT'", technology).Exec(); err != nil {
		t.Fatal(err)
	}
	ex.price = "110"
	spot0673Tick(t, r)
	if ex.places != 2 {
		t.Fatalf("3d missing OpenTime incorrectly blocked position SELL: places=%d", ex.places)
	}
	view, err := r.ledger.Accounting(pos.ID, "110")
	if err != nil || view.Position.Status != "CLOSED" {
		t.Fatalf("verified position not CLOSED after 3d SELL: %+v %v", view, err)
	}
	for _, order := range ex.orders {
		if order.Side == "SELL" && order.ClientOrderID != spotQuantExitClientID("BTCUSDT", pos.ID) {
			t.Fatalf("SELL client ID was not bound stably to the verified position: %+v", order)
		}
	}
	// The same 3d config still cannot BUY without a verified Spot market
	// OpenTime. The correction must NEVER unlock new trading by mistake.
	err = r.Run(context.Background())
	if !errors.Is(err, spotledger.ErrNeedsReview) || ex.places != 2 {
		t.Fatalf("3d BUY safety gate unexpectedly lifted: err=%v places=%d", err, ex.places)
	}
}

func TestReview0674PriorSELLOrderPreventsDuplicateAfterTimeframeEdit(t *testing.T) {
	r, ex, _, db := spot0673Fixture(t)
	spot0673Enable(t, db)
	spot0673Tick(t, r)
	pos, exists, err := r.activeSpotPosition("BTCUSDT")
	if err != nil || !exists {
		t.Fatal("missing verified entry")
	}
	// Simulate an OLD, already canceled SELL under the previous clientID
	// formula, with no SELL fills. A new timeframe must not ignore it.
	if _, err := db.Raw(`INSERT INTO spot_managed_orders(
		position_id,symbol,side,order_type,client_order_id,exchange_order_id,
		requested_qty,filled_qty,status,created_at,updated_at
	) VALUES(?,?, 'SELL','MARKET','legacy_0674_sell',NULL,
		'0.99','0','CANCELED',?,?)`,
		pos.ID, "BTCUSDT", time.Now().UnixMilli(), time.Now().UnixMilli()).Exec(); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Raw(`UPDATE spot_symbols SET technology=?
		WHERE symbol='BTCUSDT'`, `{"ema":[{"name":"ema_3d","enable":true,"kline_interval":"3d","period":5}]}`).Exec(); err != nil {
		t.Fatal(err)
	}
	ex.price = "110"
	err = r.Run(context.Background())
	if !errors.Is(err, spotledger.ErrNeedsReview) || ex.places != 1 {
		t.Fatalf("prior SELL was ignored and duplicate SELL attempted: err=%v places=%d", err, ex.places)
	}
}

func TestReview0674ExitClientIDStableAcrossClockAndTimeframe(t *testing.T) {
	one := spotQuantExitClientID("BTCUSDT", 123)
	two := spotQuantExitClientID("BTCUSDT", 124)
	if len(one) != 33 || one == two || one == spotQuantExitClientID("WBTCUSDT", 123) {
		t.Fatalf("invalid stable Spot position exit identity: %q", one)
	}
	if spotQuantExitClientID("BTCUSDT", 0) != "" {
		t.Fatal("cannot SELL nonexistent managed position")
	}
}

// Even a syntactically valid sq4 identity must cryptographically match
// the persisted symbol, BUY side, zero position ID and event time.
func TestReview0674SyntacticallyValidForgedV4OrderFailsClosed(t *testing.T) {
	r, ex, _, db := spot0673Fixture(t)
	spot0673Enable(t, db)
	event := spotCandleOpen(r.clock(), "1m")
	genuine := spotQuantClientID("BTCUSDT", "BUY", 0, "1m", event)
	forged := genuine[:12] + "0000000000000000000000"
	if forged == genuine || len(forged) != 34 {
		t.Fatalf("bad test forged id %q", forged)
	}
	// An attacker with access to the local order cache should never
	// manufacture a different BUY event by changing only the hash.
	if _, err := db.Raw(`INSERT INTO spot_managed_orders(
		position_id,symbol,side,order_type,client_order_id,exchange_order_id,
		requested_qty,filled_qty,status,created_at,updated_at
	) VALUES(999,'BTCUSDT','BUY','MARKET',?,NULL,'1','1','FILLED',1000,1000)`,
		forged).Exec(); err != nil {
		t.Fatal(err)
	}
	if _, _, ok := spotStoredBUYEvent(forged); !ok {
		t.Fatal("test ID must be structurally parseable")
	}
	err := r.Run(context.Background())
	if !errors.Is(err, spotledger.ErrNeedsReview) || ex.places != 0 {
		t.Fatalf("forged v4 BUY should not authorize new trade: err=%v places=%d", err, ex.places)
	}
}

// Strict market E2E, not just a simplified Fake evaluator: valid
// 3d indicator data and a real verified OPEN inventory may SELL
// without a separate 3d BUY-event clock provider.
func TestReview0674StrictThreeDayExitSignalUnblocked(t *testing.T) {
	r, ex, db, _, _ := spot0674StrictFixture(t)
	spot0673Tick(t, r)
	if ex.places != 1 {
		t.Fatal("expected initial 5m verified BUY")
	}
	position, exists, err := r.activeSpotPosition("BTCUSDT")
	if err != nil || !exists {
		t.Fatalf("verified inventory missing: %v", err)
	}
	tech := `{"ma":[{"name":"ma_3d","enable":true,"kline_interval":"3d","period":5}]}`
	rules := `[{"name":"new entry","type":"long","enable":true,"code":"NowPrice > 90 and ma_3d.Data[0] > 0"},{"name":"exit","type":"close_long","enable":true,"code":"ROI >= 8"}]`
	if _, err := db.Raw("UPDATE spot_symbols SET technology=?,strategy=? WHERE symbol='BTCUSDT'", tech, rules).Exec(); err != nil {
		t.Fatal(err)
	}
	ex.price = "110" // same fake Spot live quote consumed by evaluator and processor
	if err := r.Run(context.Background()); err != nil {
		t.Fatalf("strict 3d exit blocked without 3d event provider: %v", err)
	}
	if ex.places != 2 {
		t.Fatalf("strict 3d exit failed to SELL: places=%d", ex.places)
	}
	state, err := r.ledger.Accounting(position.ID, "110")
	if err != nil || state.Position.Status != "CLOSED" {
		t.Fatalf("3d strict close not CLOSED: %+v %v", state, err)
	}
	err = r.Run(context.Background())
	if !errors.Is(err, spotledger.ErrNeedsReview) || ex.places != 2 {
		t.Fatalf("3d BUY incorrectly unlocked when SELL was fixed: err=%v places=%d", err, ex.places)
	}
}
