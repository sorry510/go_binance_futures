package leadaccount

import (
	"context"
	"testing"

	"github.com/adshao/go-binance/v2/futures"
	binance "go_binance_futures/feature/api/binance"
)

func stage43RuleCase() (RiskOpenOrder, futures.Symbol, []*futures.PositionRisk, []*futures.LeverageBracket) {
	req := RiskOpenOrder{AccountID: binance.LeadAccountID, Symbol: "BTCUSDT", Side: "LONG", OrderType: "LIMIT", Quantity: 1.25,
		ReferencePrice: 100.5, Leverage: 4, MarginType: "CROSSED"}
	info := futures.Symbol{Symbol: "BTCUSDT", Status: "TRADING", QuoteAsset: "USDT", Filters: []map[string]any{
		{"filterType": "LOT_SIZE", "minQty": "0.01", "maxQty": "100", "stepSize": "0.01"},
		{"filterType": "MARKET_LOT_SIZE", "minQty": "0.01", "maxQty": "30", "stepSize": "0.01"},
		{"filterType": "PRICE_FILTER", "minPrice": "0.01", "maxPrice": "100000", "tickSize": "0.01"},
		{"filterType": "MIN_NOTIONAL", "notional": "5"},
	}}
	pos := []*futures.PositionRisk{{Symbol: "BTCUSDT", PositionSide: "LONG", MarginType: "cross", Leverage: "4"}, {Symbol: "BTCUSDT", PositionSide: "SHORT", MarginType: "cross", Leverage: "4"}}
	tiers := []*futures.LeverageBracket{{Symbol: "BTCUSDT", Brackets: []futures.Bracket{
		{InitialLeverage: 20, NotionalCap: 5000, NotionalFloor: 0}, {InitialLeverage: 5, NotionalCap: 15000, NotionalFloor: 5000},
	}}}
	return req, info, pos, tiers
}
func TestStage43LeadRulesValidateSignedSettingsAndExchangeFilters(t *testing.T) {
	r, info, pos, tier := stage43RuleCase()
	valid, err := ValidateLeadOrderRules(r, info, pos, tier, futures.MarginTypeCrossed)
	if err != nil || valid.MinNotionalUSDT != 5 || valid.MaxNotionalUSDT != 15000 || valid.MaxLeverage != 5 {
		t.Fatalf("incorrect verified rules %+v err=%v", valid, err)
	}
	r.OrderType = "MARKET"
	if _, err := ValidateLeadOrderRules(r, info, pos, tier, futures.MarginTypeCrossed); err != nil {
		t.Fatalf("market lot validation: %v", err)
	}
	r.OrderType = "LIMIT"
	r.Quantity = 1.255
	if _, err := ValidateLeadOrderRules(r, info, pos, tier, futures.MarginTypeCrossed); err == nil {
		t.Fatal("step size mismatch allowed")
	}
}
func TestStage43LeadRulesRejectMismatchAndUnknown(t *testing.T) {
	cases := []struct {
		name   string
		change func(*RiskOpenOrder, *futures.Symbol, *[]*futures.PositionRisk, *[]*futures.LeverageBracket)
	}{
		{"symbol", func(r *RiskOpenOrder, _ *futures.Symbol, _ *[]*futures.PositionRisk, _ *[]*futures.LeverageBracket) {
			r.Symbol = "ETHUSDT"
		}},
		{"account", func(r *RiskOpenOrder, _ *futures.Symbol, _ *[]*futures.PositionRisk, _ *[]*futures.LeverageBracket) {
			r.AccountID = binance.MainAccountID
		}},
		{"nontrading", func(_ *RiskOpenOrder, i *futures.Symbol, _ *[]*futures.PositionRisk, _ *[]*futures.LeverageBracket) {
			i.Status = "BREAK"
		}},
		{"lev", func(r *RiskOpenOrder, _ *futures.Symbol, _ *[]*futures.PositionRisk, _ *[]*futures.LeverageBracket) {
			r.Leverage = 21
		}},
		{"margin", func(_ *RiskOpenOrder, _ *futures.Symbol, p *[]*futures.PositionRisk, _ *[]*futures.LeverageBracket) {
			(*p)[0].MarginType = "isolated"
		}},
		{"missing_position_modes", func(_ *RiskOpenOrder, _ *futures.Symbol, p *[]*futures.PositionRisk, _ *[]*futures.LeverageBracket) {
			*p = nil
		}},
		{"price_tick", func(r *RiskOpenOrder, _ *futures.Symbol, _ *[]*futures.PositionRisk, _ *[]*futures.LeverageBracket) {
			r.ReferencePrice = 100.505
		}},
		{"market_quantity_max", func(r *RiskOpenOrder, _ *futures.Symbol, _ *[]*futures.PositionRisk, _ *[]*futures.LeverageBracket) {
			r.Quantity = 50
			r.OrderType = "MARKET"
		}},
		{"missing_filters", func(_ *RiskOpenOrder, i *futures.Symbol, _ *[]*futures.PositionRisk, _ *[]*futures.LeverageBracket) {
			i.Filters = nil
		}},
		{"bracket", func(_ *RiskOpenOrder, _ *futures.Symbol, _ *[]*futures.PositionRisk, b *[]*futures.LeverageBracket) {
			*b = nil
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r, info, pos, tier := stage43RuleCase()
			tc.change(&r, &info, &pos, &tier)
			if _, err := ValidateLeadOrderRules(r, info, pos, tier, futures.MarginTypeCrossed); err == nil {
				t.Fatal("unverified trade settings accepted")
			}
		})
	}
}

type fakeLeadRulesReader struct {
	exchange *futures.ExchangeInfo
	pos      []*futures.PositionRisk
	tier     []*futures.LeverageBracket
}

func (f *fakeLeadRulesReader) GetLeadExchangeInfoContext(context.Context) (*futures.ExchangeInfo, error) {
	return f.exchange, nil
}
func (f *fakeLeadRulesReader) GetPositionFreshContext(context.Context, binance.PositionParams) ([]*futures.PositionRisk, error) {
	return f.pos, nil
}
func (f *fakeLeadRulesReader) GetLeadLeverageBracketContext(context.Context, string) ([]*futures.LeverageBracket, error) {
	return f.tier, nil
}
func TestStage43RuleSourceStrictLeadBinding(t *testing.T) {
	req, info, pos, tier := stage43RuleCase()
	fake := &fakeLeadRulesReader{exchange: &futures.ExchangeInfo{Symbols: []futures.Symbol{info}}, pos: pos, tier: tier}
	source := &LeadRuleSource{AccountID: binance.LeadAccountID, Reader: fake}
	result, err := source.Verify(context.Background(), req, futures.MarginTypeCrossed)
	if err != nil || result.MaxNotionalUSDT != 15000 {
		t.Fatalf("read-only rule source failed: %+v %v", result, err)
	}
	source.AccountID = binance.MainAccountID
	if _, err := source.Verify(context.Background(), req, futures.MarginTypeCrossed); err == nil {
		t.Fatal("main rule reader accepted")
	}
}
