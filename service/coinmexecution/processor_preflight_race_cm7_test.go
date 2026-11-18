package coinmexecution

import (
	"context"
	"strings"
	"testing"

	"github.com/beego/beego/v2/client/orm"
	"go_binance_futures/feature"
	"go_binance_futures/feature/api/binance"
)

// CM-7 rereview F8: the Runner's last check is not enough when one of
// the subsequent signed/remote preflight calls mutates symbol settings.
type cm7PreflightGateway struct {
	*mockCM5
	stage  string
	fired  bool
	mutate func()
}

func (m *cm7PreflightGateway) attempt(stage string) {
	if !m.fired && m.stage == stage {
		m.fired = true
		m.mutate()
	}
}
func (m *cm7PreflightGateway) Spec(ctx context.Context, symbol string) (feature.CoinMPerpetualSpec, error) {
	m.attempt("Spec")
	return m.mockCM5.Spec(ctx, symbol)
}
func (m *cm7PreflightGateway) TradeSettings(ctx context.Context, symbol, side string, lev int64, margin string) error {
	m.attempt("TradeSettings")
	return m.mockCM5.TradeSettings(ctx, symbol, side, lev, margin)
}
func (m *cm7PreflightGateway) OpenAlgos(ctx context.Context, symbol string) ([]binance.CoinMAlgoOrderResponse, error) {
	m.attempt("OpenAlgos")
	return m.mockCM5.OpenAlgos(ctx, symbol)
}
func (m *cm7PreflightGateway) Query(ctx context.Context, symbol, id string) (*OrderAck, error) {
	m.attempt("Query")
	return m.mockCM5.Query(ctx, symbol, id)
}

func cm7ApplyConfigEdit(db orm.Ormer, sql string, value interface{}) error {
	if strings.HasPrefix(sql, "config ") {
		_, err := db.Raw("UPDATE config SET " + strings.TrimPrefix(sql, "config ")).Exec()
		return err
	}
	_, err := db.Raw("UPDATE delivery_symbols SET "+sql+" WHERE symbol='BTCUSD_PERP'", value).Exec()
	return err
}

func TestCM7F8SignedPreflightConfigEditsNeverPostOrPersist(t *testing.T) {
	phases := []string{"Spec", "TradeSettings", "OpenAlgos", "Query"}
	modifications := []struct {
		name, sql string
		value     interface{}
	}{
		{"disable_all_protectors", "profit='0',loss='0',tickSize='0'", nil},
		{"stop_disabled", "loss='0'", nil},
		{"take_profit_disabled", "profit='0'", nil},
		{"tick_disabled", "tickSize='0'", nil},
		{"leverage_disabled", "leverage=0", nil},
		{"change_valid_profit", "profit='9'", nil},
		{"change_valid_loss", "loss='7'", nil},
		{"change_valid_tick", "tickSize='0.2'", nil},
		{"change_valid_leverage", "leverage=5", nil},
		{"change_budget", "budget_usd='300'", nil},
		{"change_margin_mode", "marginType='ISOLATED'", nil},
		{"switch_global_OFF", "config coin_m_quant_enable=0", nil},
		{"switch_symbol_OFF", "enable=0", nil},
		{"change_strategy", "strategy='[]'", nil},
		{"change_technology", "technology='{\"ma\":[]}'", nil},
	}
	for _, phase := range phases {
		for _, tc := range modifications {
			t.Run(phase+"/"+tc.name, func(t *testing.T) {
				db := cm5Fixture(t)
				fake := newMockCM5()
				guard := &cm7PreflightGateway{mockCM5: fake, stage: phase}
				var editErr error
				guard.mutate = func() { editErr = cm7ApplyConfigEdit(db, tc.sql, tc.value) }
				p := NewProcessor(db, guard)
				_, err := p.Open(context.Background(), OpenRequest{
					Symbol: "BTCUSD_PERP", PositionSide: "LONG", Type: "MARKET",
					ClientOrderID: "cm7f8signedpreflight",
				})
				if editErr != nil {
					t.Fatalf("test change failed during %s: %v", phase, editErr)
				}
				if !guard.fired {
					t.Fatalf("%s never exercised mutation callback", phase)
				}
				if err == nil {
					t.Fatalf("%s %s change still authorized an OPEN", phase, tc.name)
				}
				if len(fake.calls) != 0 {
					t.Fatalf("%s %s sent DAPI POST %+v", phase, tc.name, fake.calls)
				}
				expectCM5(t, db, "SELECT COUNT(*) FROM coin_m_managed_orders WHERE intent='OPEN'", 0)
				expectCM5(t, db, "SELECT COUNT(*) FROM coin_m_managed_positions", 0)
			})
		}
	}
}

// When no persisted intent exists, a new explicit valid config + signal
// may still open. Blocking a stale signal must not poison the client ID.
func TestCM7F8FreshValidConfigAfterRejectedPreflightCanOpen(t *testing.T) {
	db := cm5Fixture(t)
	fake := newMockCM5()
	guard := &cm7PreflightGateway{mockCM5: fake, stage: "Spec"}
	var editErr error
	guard.mutate = func() {
		_, editErr = db.Raw("UPDATE delivery_symbols SET profit='0' WHERE symbol='BTCUSD_PERP'").Exec()
	}
	p := NewProcessor(db, guard)
	req := OpenRequest{Symbol: "BTCUSD_PERP", PositionSide: "LONG", Type: "MARKET", ClientOrderID: "cm7f8freshvalid"}
	if _, err := p.Open(context.Background(), req); err == nil {
		t.Fatal("old config POST unexpectedly succeeded")
	}
	if editErr != nil {
		t.Fatal(editErr)
	}
	if _, err := db.Raw("UPDATE delivery_symbols SET profit='8' WHERE symbol='BTCUSD_PERP'").Exec(); err != nil {
		t.Fatal(err)
	}
	guard.mutate = nil
	result, err := p.Open(context.Background(), req)
	if err != nil || result.ClientOrderID != req.ClientOrderID || len(fake.calls) != 1 {
		t.Fatalf("restored valid entry did not submit exactly once: %+v err=%v posts=%d", result, err, len(fake.calls))
	}
	expectCM5(t, db, "SELECT COUNT(*) FROM coin_m_managed_orders WHERE intent='OPEN'", 1)
}

func TestCM7F8UnsentIntentIsMarkedTerminalBeforePost(t *testing.T) {
	db := cm5Fixture(t)
	fake := newMockCM5()
	p := NewProcessor(db, fake)
	_, err := db.Raw("INSERT INTO coin_m_managed_positions(symbol,position_side,margin_asset,contract_size,entry_price,opened_contracts,remaining_contracts,realized_pnl,status,created_at,updated_at,closed_at) VALUES('BTCUSD_PERP','LONG','BTC',100,0,0,0,0,'PENDING_OPEN',1,1,0)").Exec()
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Raw("INSERT INTO coin_m_managed_orders(position_id,symbol,position_side,intent,side,order_type,client_order_id,exchange_order_id,requested_contracts,filled_contracts,status,created_at,updated_at) VALUES(1,'BTCUSD_PERP','LONG','OPEN','BUY','MARKET','cm7f8unsentlate',NULL,10,0,'PENDING_SUBMIT',1,1)").Exec()
	if err != nil {
		t.Fatal(err)
	}
	if err := p.rejectUnsentOpen("cm7f8unsentlate", 1); err != nil {
		t.Fatal(err)
	}
	expectCM5(t, db, "SELECT COUNT(*) FROM coin_m_managed_orders WHERE status='REJECTED' AND client_order_id='cm7f8unsentlate'", 1)
	expectCM5(t, db, "SELECT COUNT(*) FROM coin_m_managed_positions WHERE status='CANCELED' AND remaining_contracts=0", 1)
	if len(fake.calls) != 0 {
		t.Fatal("locally canceled unsent intent performed an exchange POST")
	}
	result, err := p.Open(context.Background(), OpenRequest{Symbol: "BTCUSD_PERP", PositionSide: "LONG", ClientOrderID: "cm7f8unsentlate", Type: "MARKET"})
	if err != nil || result.Status != "REJECTED" || len(fake.calls) != 0 {
		t.Fatalf("canceled unsent client identity was replayed: %+v %v posts=%d", result, err, len(fake.calls))
	}
}
