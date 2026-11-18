package feature

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"go_binance_futures/feature/api/binance"
	"go_binance_futures/models"

	"github.com/adshao/go-binance/v2/delivery"
	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/core/logs"
)

// CoinMPerpetualSpec is an exchange-confirmed, currently trading, USD-quoted
// inverse perpetual. Quantities are contract counts, never BTC/ETH quantity.
// CM-5 must revalidate fresh exchangeInfo before a real order.
type CoinMPerpetualSpec struct {
	Symbol         string
	ContractSize   int64
	MarginAsset    string
	ContractType   string
	TickSize       string
	MinPrice       string
	MaxPrice       string
	StepSize       string
	MinQty         string
	MaxQty         string
	MarketStepSize string
	MarketMinQty   string
	MarketMaxQty   string
}

func coinMPositiveDecimal(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" || strings.ContainsAny(s, "eE+-/") {
		return false
	}
	parts := strings.Split(s, ".")
	if len(parts) > 2 || parts[0] == "" {
		return false
	}
	for _, part := range parts {
		if part == "" {
			return false
		}
		for _, c := range part {
			if c < '0' || c > '9' {
				return false
			}
		}
	}
	r := new(big.Rat)
	_, ok := r.SetString(s)
	return ok && r.Sign() > 0
}

// The exchange may encode a disabled price bound as 0, 0.0, or 0.00000000.
func coinMNonnegativeDecimal(s string) bool {
	if coinMPositiveDecimal(s) {
		return true
	}
	if strings.ContainsAny(s, "eE+-/") {
		return false
	}
	r, ok := new(big.Rat).SetString(s)
	return ok && r.Sign() == 0
}

func coinMDecimalLessOrEqual(a, b string) bool {
	ar, _ := new(big.Rat).SetString(a)
	br, _ := new(big.Rat).SetString(b)
	return ar.Cmp(br) <= 0
}

// Read dynamic map filters without SDK's unchecked type assertions. Binance
// occasionally adds new filter types; unknown filters are safely ignored.
func coinMFilter(values []map[string]any, kind string) (map[string]any, bool) {
	for _, item := range values {
		if name, ok := item["filterType"].(string); ok && name == kind {
			return item, true
		}
	}
	return nil, false
}

func coinMFilterValue(filter map[string]any, field string) string {
	if value, ok := filter[field].(string); ok {
		return strings.TrimSpace(value)
	}
	return ""
}

// ParseCoinMPerpetualSpec parses only metadata. Ineligible contracts are
// filtered by IsCoinMTradingPerpetual before calling this function.
func IsCoinMTradingPerpetual(s delivery.Symbol) bool {
	return s.ContractType == "PERPETUAL" && s.ContractStatus == "TRADING" &&
		s.QuoteAsset == "USD" && strings.HasSuffix(s.Symbol, "_PERP")
}

func ParseCoinMPerpetualSpec(s delivery.Symbol) (CoinMPerpetualSpec, error) {
	var spec CoinMPerpetualSpec
	if !IsCoinMTradingPerpetual(s) {
		return spec, errors.New("COIN-M symbol is not a trading USD perpetual")
	}
	if s.ContractSize <= 0 || strings.TrimSpace(s.MarginAsset) == "" {
		return spec, fmt.Errorf("COIN-M %s missing contract size or margin asset", s.Symbol)
	}
	price, okPrice := coinMFilter(s.Filters, "PRICE_FILTER")
	lot, okLot := coinMFilter(s.Filters, "LOT_SIZE")
	market, okMarket := coinMFilter(s.Filters, "MARKET_LOT_SIZE")
	if !okPrice || !okLot || !okMarket {
		return spec, fmt.Errorf("COIN-M %s incomplete exchange filters", s.Symbol)
	}
	spec = CoinMPerpetualSpec{
		Symbol: s.Symbol, ContractSize: int64(s.ContractSize),
		MarginAsset: strings.TrimSpace(s.MarginAsset), ContractType: s.ContractType,
		TickSize:       coinMFilterValue(price, "tickSize"),
		MinPrice:       coinMFilterValue(price, "minPrice"),
		MaxPrice:       coinMFilterValue(price, "maxPrice"),
		StepSize:       coinMFilterValue(lot, "stepSize"),
		MinQty:         coinMFilterValue(lot, "minQty"),
		MaxQty:         coinMFilterValue(lot, "maxQty"),
		MarketStepSize: coinMFilterValue(market, "stepSize"),
		MarketMinQty:   coinMFilterValue(market, "minQty"),
		MarketMaxQty:   coinMFilterValue(market, "maxQty"),
	}
	for _, item := range []string{spec.TickSize, spec.StepSize, spec.MinQty, spec.MaxQty,
		spec.MarketStepSize, spec.MarketMinQty, spec.MarketMaxQty} {
		if !coinMPositiveDecimal(item) {
			return CoinMPerpetualSpec{}, fmt.Errorf("COIN-M %s invalid PRICE/LOT/MARKET filter", s.Symbol)
		}
	}
	if !coinMDecimalLessOrEqual(spec.MinQty, spec.MaxQty) ||
		!coinMDecimalLessOrEqual(spec.MarketMinQty, spec.MarketMaxQty) {
		return CoinMPerpetualSpec{}, fmt.Errorf("COIN-M %s invalid quantity limits", s.Symbol)
	}
	// minPrice/maxPrice may legally be 0 when the bound is disabled.
	for _, item := range []string{spec.MinPrice, spec.MaxPrice} {
		if !coinMNonnegativeDecimal(item) {
			return CoinMPerpetualSpec{}, fmt.Errorf("COIN-M %s invalid price bounds", s.Symbol)
		}
	}
	return spec, nil
}

// ValidateCoinMOrderFilters checks raw quantity as CONTRACT COUNT. This is
// a pure preflight primitive: it never submits an order or modifies an account.
// An order must still re-fetch fresh symbol metadata in CM-5.
func ValidateCoinMOrderFilters(spec CoinMPerpetualSpec, contracts, limitPrice string, market bool) error {
	if !coinMPositiveDecimal(contracts) {
		return errors.New("COIN-M contract quantity must be positive")
	}
	minQty, maxQty, step := spec.MinQty, spec.MaxQty, spec.StepSize
	if market {
		minQty, maxQty, step = spec.MarketMinQty, spec.MarketMaxQty, spec.MarketStepSize
	}
	for _, value := range []string{minQty, maxQty, step} {
		if !coinMPositiveDecimal(value) {
			return errors.New("COIN-M quantity filter is unavailable")
		}
	}
	q, _ := new(big.Rat).SetString(contracts)
	min, _ := new(big.Rat).SetString(minQty)
	max, _ := new(big.Rat).SetString(maxQty)
	increment, _ := new(big.Rat).SetString(step)
	if q.Cmp(min) < 0 || q.Cmp(max) > 0 || !new(big.Rat).Quo(q, increment).IsInt() {
		return fmt.Errorf("COIN-M contract count violates exchange quantity filters")
	}
	if market {
		return nil
	}
	if !coinMPositiveDecimal(limitPrice) || !coinMPositiveDecimal(spec.TickSize) {
		return errors.New("COIN-M limit price/tick is unavailable")
	}
	price, _ := new(big.Rat).SetString(limitPrice)
	tick, _ := new(big.Rat).SetString(spec.TickSize)
	if !new(big.Rat).Quo(price, tick).IsInt() {
		return errors.New("COIN-M limit price violates tick size")
	}
	if minPrice, ok := new(big.Rat).SetString(spec.MinPrice); !ok || price.Cmp(minPrice) < 0 {
		return errors.New("COIN-M limit price below exchange minimum")
	}
	if maxPrice, ok := new(big.Rat).SetString(spec.MaxPrice); !ok ||
		(maxPrice.Sign() > 0 && price.Cmp(maxPrice) > 0) {
		return errors.New("COIN-M limit price exceeds exchange maximum")
	}
	return nil
}

// RefreshCoinMPerpetualSymbols validates the complete response BEFORE the
// first database write, so malformed metadata never partially updates coins.
// It changes only exchange metadata; existing enable/strategy/USDT fields
// remain operator-owned. Newly discovered symbols start enable=0, budget=0.
func RefreshCoinMPerpetualSymbols(ctx context.Context, db orm.Ormer, info *delivery.ExchangeInfo) (int, error) {
	if db == nil || info == nil {
		return 0, errors.New("COIN-M symbol DB or exchangeInfo unavailable")
	}
	var specs []CoinMPerpetualSpec
	seen := make(map[string]bool)
	for _, symbol := range info.Symbols {
		if !IsCoinMTradingPerpetual(symbol) {
			continue
		}
		spec, err := ParseCoinMPerpetualSpec(symbol)
		if err != nil {
			return 0, err
		}
		if seen[spec.Symbol] {
			return 0, fmt.Errorf("duplicate COIN-M symbol %s", spec.Symbol)
		}
		seen[spec.Symbol] = true
		specs = append(specs, spec)
	}
	if len(specs) == 0 {
		return 0, errors.New("COIN-M exchangeInfo has no valid perpetual contracts")
	}
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	tx, err := db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	for _, spec := range specs {
		if err := ctx.Err(); err != nil {
			return 0, err
		}
		var current models.DeliverySymbols
		err := tx.QueryTable(new(models.DeliverySymbols)).Filter("symbol", spec.Symbol).One(&current)
		switch {
		case errors.Is(err, orm.ErrNoRows):
			_, err = tx.Insert(&models.DeliverySymbols{
				Symbol: spec.Symbol, Enable: 0, Leverage: 3, MarginType: "CROSSED",
				TickSize: spec.TickSize, StepSize: spec.StepSize,
				ContractSize: spec.ContractSize, MarginAsset: spec.MarginAsset,
				ContractType: spec.ContractType, Usdt: "0", Profit: "20", Loss: "20",
				Type: "PERP",
			})
		case err == nil:
			_, err = tx.QueryTable(new(models.DeliverySymbols)).Filter("id", current.ID).Update(orm.Params{
				"TickSize": spec.TickSize, "StepSize": spec.StepSize,
				"ContractSize": spec.ContractSize, "MarginAsset": spec.MarginAsset,
				"ContractType": spec.ContractType,
			})
		}
		if err != nil {
			return 0, fmt.Errorf("persist COIN-M %s metadata: %w", spec.Symbol, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit COIN-M metadata: %w", err)
	}
	return len(specs), nil
}

func UpdateDeliverySymbolsTradePrecision() {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	info, err := binance.GetDeliveryExchangeInfoContext(ctx)
	if err != nil {
		logs.Warning("COIN-M exchangeInfo unavailable: ", err)
		return
	}
	count, err := RefreshCoinMPerpetualSymbols(ctx, orm.NewOrm(), info)
	if err != nil {
		logs.Warning("COIN-M symbol refresh skipped: ", err)
		return
	}
	logs.Info("COIN-M perpetual symbols refreshed: ", count)
}
