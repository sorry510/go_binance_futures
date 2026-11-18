package coinmquant

import (
	"testing"

	"github.com/beego/beego/v2/client/orm"
)

// Permanent CM-4 rereview F1 probe. Both the Fill and its parent order may
// be internally self-consistent but belong to a different COIN-M contract or
// hedge side than their claimed Managed position. Never label such evidence
// verified; never produce an unrealized / total PnL or asset total.
func TestCM4CrossSymbolAndPositionSideFillMustBeRejected(t *testing.T) {
	cases := []struct {
		name          string
		breakEvidence func(t *testing.T, o orm.Ormer)
	}{
		{"order_and_fill_other_symbol", func(t *testing.T, o orm.Ormer) {
			execCM4Corruption(t, o, "UPDATE coin_m_managed_orders SET symbol='ETHUSD_PERP' WHERE position_id=1")
			execCM4Corruption(t, o, "UPDATE coin_m_trade_fills SET symbol='ETHUSD_PERP' WHERE position_id=1")
		}},
		{"order_and_fill_other_hedge_side", func(t *testing.T, o orm.Ormer) {
			execCM4Corruption(t, o, "UPDATE coin_m_managed_orders SET position_side='SHORT' WHERE position_id=1")
			execCM4Corruption(t, o, "UPDATE coin_m_trade_fills SET position_side='SHORT' WHERE position_id=1")
		}},
		{"other_contract_and_hedge_side_both_self_consistent", func(t *testing.T, o orm.Ormer) {
			execCM4Corruption(t, o, "UPDATE coin_m_managed_orders SET symbol='ETHUSD_PERP',position_side='SHORT' WHERE position_id=1")
			execCM4Corruption(t, o, "UPDATE coin_m_trade_fills SET symbol='ETHUSD_PERP',position_side='SHORT' WHERE position_id=1")
		}},
		{"other_symbol_and_hedge_side_with_valid_short_buy_sell", func(t *testing.T, o orm.Ormer) {
			execCM4Corruption(t, o, "UPDATE coin_m_managed_orders SET symbol='ETHUSD_PERP',position_side='SHORT',side=CASE WHEN intent='OPEN' THEN 'SELL' ELSE 'BUY' END WHERE position_id=1")
			execCM4Corruption(t, o, "UPDATE coin_m_trade_fills SET symbol='ETHUSD_PERP',position_side='SHORT',side=CASE WHEN order_id=11 THEN 'SELL' ELSE 'BUY' END WHERE position_id=1")
		}},
		{"fill_other_symbol_only", func(t *testing.T, o orm.Ormer) {
			execCM4Corruption(t, o, "UPDATE coin_m_trade_fills SET symbol='ETHUSD_PERP' WHERE order_id=11")
		}},
		{"order_other_symbol_only", func(t *testing.T, o orm.Ormer) {
			execCM4Corruption(t, o, "UPDATE coin_m_managed_orders SET symbol='ETHUSD_PERP' WHERE id=11")
		}},
		{"fill_other_hedge_side_only", func(t *testing.T, o orm.Ormer) {
			execCM4Corruption(t, o, "UPDATE coin_m_trade_fills SET position_side='SHORT' WHERE id=1")
		}},
		{"order_other_hedge_side_only", func(t *testing.T, o orm.Ormer) {
			execCM4Corruption(t, o, "UPDATE coin_m_managed_orders SET position_side='SHORT' WHERE id=11")
		}},
		{"order_side_mismatch", func(t *testing.T, o orm.Ormer) {
			execCM4Corruption(t, o, "UPDATE coin_m_managed_orders SET side='SELL' WHERE id=11")
		}},
		{"orphan_fill_missing_order", func(t *testing.T, o orm.Ormer) {
			execCM4Corruption(t, o, "DELETE FROM coin_m_managed_orders WHERE id=11")
		}},
		{"order_claims_different_managed_position", func(t *testing.T, o orm.Ormer) {
			execCM4Corruption(t, o, "UPDATE coin_m_managed_orders SET position_id=999 WHERE id=11")
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			o := cm4Fixture(t)
			cm4Position(t, o, 1, "BTCUSD_PERP", "LONG", "BTC", "100", "50000", "10", "5", "0.000909090909090909", "OPEN")
			cm4Fill(t, o, 1, 11, "BTCUSD_PERP", "LONG", "BTC", "OPEN", "10", "0", "0.00001", "BTC")
			cm4Fill(t, o, 1, 12, "BTCUSD_PERP", "LONG", "BTC", "CLOSE", "5", "0.000909090909090909", "0.00001", "BTC")
			valid, err := PositionSummaries(o, LedgerFilter{}, cm4Marks())
			if err != nil || len(valid.List) != 1 || valid.List[0].PnLStatus != "verified" {
				t.Fatalf("valid baseline should be verified: %+v %v", valid, err)
			}
			tc.breakEvidence(t, o)
			page, err := PositionSummaries(o, LedgerFilter{}, cm4Marks())
			if err != nil {
				t.Fatal(err)
			}
			if page.Total != 1 || len(page.List) != 1 || len(page.Totals) != 1 {
				t.Fatalf("unexpected result shape: %+v", page)
			}
			s := page.List[0]
			a := page.Totals[0]
			if s.PnLStatus != "invalid" || s.UnrealizedPnL != nil ||
				s.TotalPnL != nil || s.TotalUSD != nil || s.RealizedUSD != nil ||
				a.Invalid != 1 || a.Verified != 0 || a.TotalPnL != nil ||
				a.UnrealizedPnL != nil || a.TotalUSD != nil || a.RealizedUSD != nil {
				t.Fatalf("invalid Fill polluted verified PnL totals: status=%s warning=%s cycle=%+v total=%+v",
					s.PnLStatus, s.Warning, s, a)
			}
		})
	}
}
func execCM4Corruption(t *testing.T, o orm.Ormer, query string) {
	t.Helper()
	if _, err := o.Raw(query).Exec(); err != nil {
		t.Fatalf("corrupt isolated SQLite fixture: %v", err)
	}
}
